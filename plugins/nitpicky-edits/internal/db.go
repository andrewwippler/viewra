package internal

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	_ "github.com/lib/pq"
)

// openDB opens a database connection based on the config.
func openDB(cfg *Config) (*sql.DB, error) {
	switch strings.ToLower(cfg.DBDriver) {
	case "sqlite", "sqlite3":
		db, err := sql.Open("sqlite3", cfg.DBDataSource)
		if err != nil {
			return nil, fmt.Errorf("open sqlite: %w", err)
		}
		db.SetMaxOpenConns(1)
		return db, nil
	case "postgres", "postgresql":
		db, err := sql.Open("postgres", cfg.DBDataSource)
		if err != nil {
			return nil, fmt.Errorf("open postgres: %w", err)
		}
		return db, nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.DBDriver)
	}
}

// identifyMovie sets external IDs for a movie.
func identifyMovie(ctx context.Context, db *sql.DB, movieID int64, req IdentifyMovieRequest, driver string) error {
	var sets []string
	var args []interface{}
	argIdx := 1

	if req.IMDbID != "" {
		sets = append(sets, fmt.Sprintf("imdb_id = $%d", argIdx))
		args = append(args, req.IMDbID)
		argIdx++
	}
	if req.TMDbID != nil {
		sets = append(sets, fmt.Sprintf("tmdb_id = $%d", argIdx))
		args = append(args, *req.TMDbID)
		argIdx++
	}

	if len(sets) == 0 {
		return fmt.Errorf("no external IDs provided")
	}

	// Convert $N placeholders to ? for SQLite
	query := fmt.Sprintf("UPDATE media_items SET %s WHERE id = $%d", strings.Join(sets, ", "), argIdx)
	args = append(args, movieID)
	query = rebind(query, driver)

	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update movie external IDs: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("movie not found: %d", movieID)
	}
	return nil
}

// identifyTVShow sets external IDs for a TV show.
func identifyTVShow(ctx context.Context, db *sql.DB, showID int64, req IdentifyTVShowRequest, driver string) error {
	if req.IMDbID == "" && req.TVDbID == nil && req.TMDbID == nil {
		return fmt.Errorf("no external IDs provided")
	}

	var sets []string
	var args []interface{}
	argIdx := 1

	if req.IMDbID != "" {
		sets = append(sets, fmt.Sprintf("imdb_id = $%d", argIdx))
		args = append(args, req.IMDbID)
		argIdx++
	}
	if req.TVDbID != nil {
		sets = append(sets, fmt.Sprintf("tvdb_id = $%d", argIdx))
		args = append(args, *req.TVDbID)
		argIdx++
	}
	if req.TMDbID != nil {
		sets = append(sets, fmt.Sprintf("tmdb_id = $%d", argIdx))
		args = append(args, *req.TMDbID)
		argIdx++
	}

	query := fmt.Sprintf("UPDATE tv_shows SET %s WHERE id = $%d", strings.Join(sets, ", "), argIdx)
	args = append(args, showID)
	query = rebind(query, driver)

	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update TV show external IDs: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("TV show not found: %d", showID)
	}
	return nil
}

// rebind converts PostgreSQL $N placeholders to ? for SQLite compatibility.
// For PostgreSQL, it returns the query unchanged (PostgreSQL uses $N natively).
func rebind(query string, driver string) string {
	switch driver {
	case "postgres", "postgresql":
		return query
	default:
		// Convert $N to ? for SQLite
		buf := make([]byte, 0, len(query))
		i := 0
		for j := 0; j < len(query); j++ {
			if query[j] == '$' {
				buf = append(buf, query[i:j]...)
				// Skip past the digit(s)
				k := j + 1
				for k < len(query) && query[k] >= '0' && query[k] <= '9' {
					k++
				}
				buf = append(buf, '?')
				i = k
			}
		}
		buf = append(buf, query[i:]...)
		return string(buf)
	}
}
