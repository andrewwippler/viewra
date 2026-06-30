package internal

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
}

func TestNewPlugin(t *testing.T) {
	p := NewPlugin(newTestLogger())
	if p == nil {
		t.Fatal("NewPlugin() returned nil")
	}
	if p.logger == nil {
		t.Error("logger should be set")
	}
	if p.config == nil {
		t.Error("config should be set")
	}
	if !p.config.EditModeRequired {
		t.Error("EditModeRequired should default to true")
	}
	if p.db != nil {
		t.Error("db should be nil before Initialize")
	}
}

func TestPlugin_GetCapabilities(t *testing.T) {
	p := NewPlugin(newTestLogger())
	caps := p.GetCapabilities()
	if len(caps.MediaTypes) != 3 {
		t.Errorf("expected 3 media types, got %d", len(caps.MediaTypes))
	}
	if !caps.IsLocal {
		t.Error("IsLocal should be true")
	}
}

func TestPlugin_IsConfigured(t *testing.T) {
	p := NewPlugin(newTestLogger())
	if p.IsConfigured() {
		t.Error("IsConfigured should return false before Initialize")
	}
}

func TestPlugin_GetRoutes(t *testing.T) {
	p := NewPlugin(newTestLogger())
	routes := p.GetRoutes()
	if len(routes) != 12 {
		t.Errorf("expected 12 routes, got %d", len(routes))
	}
}

func TestPlugin_Shutdown(t *testing.T) {
	p := NewPlugin(newTestLogger())
	f := t.TempDir() + "/test.db"
	db, err := sql.Open("sqlite3", f)
	if err != nil {
		t.Fatal(err)
	}
	p.db = db
	if err := p.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}
	// DB should be closed after shutdown
	if err := db.Ping(); err == nil {
		t.Error("db should be closed after Shutdown")
	}
}

func TestPlugin_Configure(t *testing.T) {
	p := NewPlugin(newTestLogger())

	cfg := Config{EditModeRequired: false}
	data, _ := json.Marshal(cfg)
	if err := p.Configure(data); err != nil {
		t.Fatalf("Configure() error = %v", err)
	}

	if p.config.EditModeRequired {
		t.Error("EditModeRequired should be false after Configure")
	}
}

func TestPlugin_GetSettingsSchema(t *testing.T) {
	p := NewPlugin(newTestLogger())
	schema, err := p.GetSettingsSchema()
	if err != nil {
		t.Fatalf("GetSettingsSchema() error = %v", err)
	}
	if len(schema) == 0 {
		t.Error("expected non-empty schema")
	}
	var parsed interface{}
	if err := json.Unmarshal(schema, &parsed); err != nil {
		t.Errorf("schema should be valid JSON: %v", err)
	}
}

func TestParseInt64(t *testing.T) {
	tests := []struct {
		input string
		want  int64
		fails bool
	}{
		{"0", 0, false},
		{"123", 123, false},
		{"9999999999999", 9999999999999, false},
		{"abc", 0, true},
		{"12.5", 0, true},
		{"-5", 0, true},
	}

	for _, tt := range tests {
		got, err := parseInt64(tt.input)
		if tt.fails {
			if err == nil {
				t.Errorf("parseInt64(%q) should error", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("parseInt64(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("parseInt64(%q) = %d, want %d", tt.input, got, tt.want)
			}
		}
	}
}

func TestRebind(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"SELECT * FROM media_items WHERE id = $1", "SELECT * FROM media_items WHERE id = ?"},
		{"UPDATE media_items SET imdb_id = $1, tmdb_id = $2 WHERE id = $3", "UPDATE media_items SET imdb_id = ?, tmdb_id = ? WHERE id = ?"},
		{"SELECT * FROM tv_shows WHERE id = $1 AND name = $2", "SELECT * FROM tv_shows WHERE id = ? AND name = ?"},
		{"SELECT 1", "SELECT 1"},
		{"", ""},
	}

	for _, tt := range tests {
		got := rebind(tt.input, "sqlite3")
		if got != tt.want {
			t.Errorf("rebind(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSettingsSchema(t *testing.T) {
	schema, err := SettingsSchema()
	if err != nil {
		t.Fatalf("SettingsSchema() error = %v", err)
	}
	if len(schema) == 0 {
		t.Error("expected non-empty schema")
	}
	var parsed interface{}
	if err := json.Unmarshal(schema, &parsed); err != nil {
		t.Errorf("schema should be valid JSON: %v", err)
	}
}

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	f := t.TempDir() + "/test.db"
	db, err := sql.Open("sqlite3", f)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	schema := `
	CREATE TABLE IF NOT EXISTS media_items (
		id INTEGER PRIMARY KEY,
		imdb_id TEXT,
		tmdb_id INTEGER
	);
	CREATE TABLE IF NOT EXISTS tv_shows (
		id INTEGER PRIMARY KEY,
		imdb_id TEXT,
		tvdb_id INTEGER,
		tmdb_id INTEGER
	);
	CREATE TABLE IF NOT EXISTS tv_seasons (
		id INTEGER PRIMARY KEY,
		show_id INTEGER NOT NULL
	);
	CREATE TABLE IF NOT EXISTS tv_episodes (
		id INTEGER PRIMARY KEY,
		season_id INTEGER NOT NULL,
		show_id INTEGER NOT NULL
	);
	CREATE TABLE IF NOT EXISTS watch_progress (
		media_id INTEGER PRIMARY KEY,
		is_watched INTEGER NOT NULL DEFAULT 0,
		position REAL NOT NULL DEFAULT 0,
		duration REAL NOT NULL DEFAULT 0,
		updated_at DATETIME NOT NULL
	);
	CREATE TABLE IF NOT EXISTS ratings (
		entity_type TEXT NOT NULL,
		entity_id INTEGER NOT NULL
	);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}
	return db
}

func TestIdentifyMovie(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.Exec("INSERT INTO media_items (id) VALUES (1)")
	if err != nil {
		t.Fatal(err)
	}

	err = identifyMovie(context.Background(), db, 1, IdentifyMovieRequest{
		IMDbID: "tt1234567",
		TMDbID: intPtr(550),
	}, "sqlite3")
	if err != nil {
		t.Fatalf("identifyMovie() error = %v", err)
	}

	var imdbID string
	var tmdbID *int
	row := db.QueryRow("SELECT imdb_id, tmdb_id FROM media_items WHERE id = 1")
	if err := row.Scan(&imdbID, &tmdbID); err != nil {
		t.Fatal(err)
	}
	if imdbID != "tt1234567" {
		t.Errorf("imdb_id = %q", imdbID)
	}
	if tmdbID == nil || *tmdbID != 550 {
		t.Errorf("tmdb_id = %v", tmdbID)
	}
}

func TestIdentifyMovie_NotFound(t *testing.T) {
	db := setupTestDB(t)
	err := identifyMovie(context.Background(), db, 999, IdentifyMovieRequest{
		IMDbID: "tt1234567",
	}, "sqlite3")
	if err == nil {
		t.Error("expected error for non-existent movie")
	}
}

func TestIdentifyTVShow(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.Exec("INSERT INTO tv_shows (id) VALUES (1)")
	if err != nil {
		t.Fatal(err)
	}

	err = identifyTVShow(context.Background(), db, 1, IdentifyTVShowRequest{
		IMDbID: "tt1234567",
		TVDbID: intPtr(123),
		TMDbID: intPtr(456),
	}, "sqlite3")
	if err != nil {
		t.Fatalf("identifyTVShow() error = %v", err)
	}

	var imdbID string
	var tvdbID, tmdbID *int
	row := db.QueryRow("SELECT imdb_id, tvdb_id, tmdb_id FROM tv_shows WHERE id = 1")
	if err := row.Scan(&imdbID, &tvdbID, &tmdbID); err != nil {
		t.Fatal(err)
	}
	if imdbID != "tt1234567" {
		t.Errorf("imdb_id = %q", imdbID)
	}
	if tvdbID == nil || *tvdbID != 123 {
		t.Errorf("tvdb_id = %v", tvdbID)
	}
	if tmdbID == nil || *tmdbID != 456 {
		t.Errorf("tmdb_id = %v", tmdbID)
	}
}

func TestIdentifyTVShow_NotFound(t *testing.T) {
	db := setupTestDB(t)
	err := identifyTVShow(context.Background(), db, 999, IdentifyTVShowRequest{
		IMDbID: "tt1234567",
	}, "sqlite3")
	if err == nil {
		t.Error("expected error for non-existent show")
	}
}

func TestDeleteTVEpisode(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.Exec("INSERT INTO tv_episodes (id, season_id, show_id) VALUES (1, 1, 1)")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec("INSERT INTO watch_progress (media_id, is_watched, position, duration, updated_at) VALUES (1, 1, 0, 0, datetime('now'))")
	if err != nil {
		t.Fatal(err)
	}

	err = deleteTVEpisode(context.Background(), db, 1, "sqlite3")
	if err != nil {
		t.Fatalf("deleteTVEpisode() error = %v", err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM tv_episodes WHERE id = 1").Scan(&count)
	if count != 0 {
		t.Error("episode should be deleted")
	}
	db.QueryRow("SELECT COUNT(*) FROM watch_progress WHERE media_id = 1").Scan(&count)
	if count != 0 {
		t.Error("watch_progress should be deleted")
	}
}

func TestDeleteTVEpisode_NotFound(t *testing.T) {
	db := setupTestDB(t)
	err := deleteTVEpisode(context.Background(), db, 999, "sqlite3")
	if err == nil {
		t.Error("expected error for non-existent episode")
	}
}

func TestDeleteTVSeason(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.Exec("INSERT INTO tv_seasons (id, show_id) VALUES (1, 1)")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec("INSERT INTO tv_episodes (id, season_id, show_id) VALUES (1, 1, 1), (2, 1, 1)")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec("INSERT INTO watch_progress (media_id, is_watched, position, duration, updated_at) VALUES (1, 1, 0, 0, datetime('now')), (2, 0, 0, 0, datetime('now'))")
	if err != nil {
		t.Fatal(err)
	}

	err = deleteTVSeason(context.Background(), db, 1, "sqlite3")
	if err != nil {
		t.Fatalf("deleteTVSeason() error = %v", err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM tv_seasons WHERE id = 1").Scan(&count)
	if count != 0 {
		t.Error("season should be deleted")
	}
	db.QueryRow("SELECT COUNT(*) FROM tv_episodes WHERE season_id = 1").Scan(&count)
	if count != 0 {
		t.Error("episodes should be deleted")
	}
}

func TestDeleteTVSeason_NotFound(t *testing.T) {
	db := setupTestDB(t)
	err := deleteTVSeason(context.Background(), db, 999, "sqlite3")
	if err == nil {
		t.Error("expected error for non-existent season")
	}
}

func TestDeleteTVShow(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.Exec("INSERT INTO tv_shows (id) VALUES (1)")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec("INSERT INTO tv_seasons (id, show_id) VALUES (1, 1)")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec("INSERT INTO tv_episodes (id, season_id, show_id) VALUES (1, 1, 1)")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec("INSERT INTO watch_progress (media_id, is_watched, position, duration, updated_at) VALUES (1, 1, 0, 0, datetime('now'))")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec("INSERT INTO ratings (entity_type, entity_id) VALUES ('tv_show', 1)")
	if err != nil {
		t.Fatal(err)
	}

	err = deleteTVShow(context.Background(), db, 1, "sqlite3")
	if err != nil {
		t.Fatalf("deleteTVShow() error = %v", err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM tv_shows WHERE id = 1").Scan(&count)
	if count != 0 {
		t.Error("show should be deleted")
	}
	db.QueryRow("SELECT COUNT(*) FROM watch_progress WHERE media_id IN (SELECT id FROM tv_episodes WHERE show_id = 1)").Scan(&count)
	if count != 0 {
		t.Error("watch_progress should be deleted")
	}
	db.QueryRow("SELECT COUNT(*) FROM ratings WHERE entity_type = 'tv_show' AND entity_id = 1").Scan(&count)
	if count != 0 {
		t.Error("ratings should be deleted")
	}
}

func TestDeleteTVShow_NotFound(t *testing.T) {
	db := setupTestDB(t)
	err := deleteTVShow(context.Background(), db, 999, "sqlite3")
	if err == nil {
		t.Error("expected error for non-existent show")
	}
}

func TestMarkWatched(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.Exec("INSERT INTO tv_episodes (id, season_id, show_id) VALUES (1, 1, 1)")
	if err != nil {
		t.Fatal(err)
	}

	err = markWatched(context.Background(), db, 1)
	if err != nil {
		t.Fatalf("markWatched() error = %v", err)
	}

	var isWatched int
	db.QueryRow("SELECT is_watched FROM watch_progress WHERE media_id = 1").Scan(&isWatched)
	if isWatched != 1 {
		t.Errorf("is_watched = %d, want 1", isWatched)
	}
}

func TestMarkUnwatched(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.Exec("INSERT INTO tv_episodes (id, season_id, show_id) VALUES (1, 1, 1)")
	if err != nil {
		t.Fatal(err)
	}

	err = markUnwatched(context.Background(), db, 1)
	if err != nil {
		t.Fatalf("markUnwatched() error = %v", err)
	}

	var isWatched int
	db.QueryRow("SELECT is_watched FROM watch_progress WHERE media_id = 1").Scan(&isWatched)
	if isWatched != 0 {
		t.Errorf("is_watched = %d, want 0", isWatched)
	}
}

func TestMarkSeasonWatched(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.Exec("INSERT INTO tv_seasons (id, show_id) VALUES (1, 1)")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec("INSERT INTO tv_episodes (id, season_id, show_id) VALUES (1, 1, 1), (2, 1, 1)")
	if err != nil {
		t.Fatal(err)
	}

	// Collect episode IDs first to avoid connection contention with MaxOpenConns
	var ids []int64
	rows, err := db.Query("SELECT id FROM tv_episodes WHERE season_id = 1")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id int64
		rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close()

	for _, id := range ids {
		if err := markWatched(context.Background(), db, id); err != nil {
			t.Fatalf("markWatched(%d) error = %v", id, err)
		}
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM watch_progress WHERE is_watched = 1").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 watched entries, got %d", count)
	}
}

func TestMarkSeasonUnwatched(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.Exec("INSERT INTO tv_seasons (id, show_id) VALUES (1, 1)")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec("INSERT INTO tv_episodes (id, season_id, show_id) VALUES (1, 1, 1), (2, 1, 1)")
	if err != nil {
		t.Fatal(err)
	}

	var ids []int64
	rows, err := db.Query("SELECT id FROM tv_episodes WHERE season_id = 1")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id int64
		rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close()

	for _, id := range ids {
		if _, err := db.Exec("INSERT INTO watch_progress (media_id, is_watched, position, duration, updated_at) VALUES (?, 1, 0, 0, datetime('now'))", id); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range ids {
		if err := markUnwatched(context.Background(), db, id); err != nil {
			t.Fatalf("markUnwatched(%d) error = %v", id, err)
		}
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM watch_progress WHERE is_watched = 1").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 watched entries, got %d", count)
	}
}

func TestOpenDB_SQLite(t *testing.T) {
	f := t.TempDir() + "/test.db"
	db, err := openDB(&Config{
		DBDriver:     "sqlite3",
		DBDataSource: f,
	})
	if err != nil {
		t.Fatalf("openDB() error = %v", err)
	}
	defer db.Close()

	if db == nil {
		t.Error("db should not be nil")
	}
}

func TestRebindNoOp(t *testing.T) {
	// rebind with no $N should return unchanged
	input := "SELECT * FROM foo WHERE id = ?"
	got := rebind(input, "sqlite3")
	if got != input {
		t.Errorf("rebind(%q) = %q, want unchanged", input, got)
	}
}

func intPtr(i int) *int {
	return &i
}
