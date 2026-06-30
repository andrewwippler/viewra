package execution

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mantonx/viewra/internal/domain/livetv"
)

type mockChannelRepo struct {
	channels      map[int64]*livetv.Channel
	listByLibFn   func(ctx context.Context, libraryID int64) ([]*livetv.Channel, error)
	nextID        int64
}

func newMockChannelRepo() *mockChannelRepo {
	return &mockChannelRepo{
		channels: make(map[int64]*livetv.Channel),
		nextID:   1,
	}
}

func (r *mockChannelRepo) Create(ctx context.Context, ch *livetv.Channel) error {
	ch.ID = r.nextID
	r.nextID++
	r.channels[ch.ID] = ch
	return nil
}

func (r *mockChannelRepo) GetByID(ctx context.Context, id int64) (*livetv.Channel, error) {
	ch, ok := r.channels[id]
	if !ok {
		return nil, nil
	}
	return ch, nil
}

func (r *mockChannelRepo) ListByLibrary(ctx context.Context, libraryID int64) ([]*livetv.Channel, error) {
	if r.listByLibFn != nil {
		return r.listByLibFn(ctx, libraryID)
	}
	var result []*livetv.Channel
	for _, ch := range r.channels {
		if ch.LibraryID == libraryID {
			result = append(result, ch)
		}
	}
	return result, nil
}

func (r *mockChannelRepo) Upsert(ctx context.Context, ch *livetv.Channel) error {
	if ch.ID == 0 {
		ch.ID = r.nextID
		r.nextID++
	}
	r.channels[ch.ID] = ch
	return nil
}

func (r *mockChannelRepo) Delete(ctx context.Context, id int64) error {
	delete(r.channels, id)
	return nil
}

func (r *mockChannelRepo) DeleteByLibrary(ctx context.Context, libraryID int64) error {
	for id, ch := range r.channels {
		if ch.LibraryID == libraryID {
			delete(r.channels, id)
		}
	}
	return nil
}

type mockProgramRepo struct {
	programs map[int64]*livetv.Program
	nextID   int64
}

func newMockProgramRepo() *mockProgramRepo {
	return &mockProgramRepo{
		programs: make(map[int64]*livetv.Program),
		nextID:   1,
	}
}

func (r *mockProgramRepo) Create(ctx context.Context, p *livetv.Program) error {
	p.ID = r.nextID
	r.nextID++
	r.programs[p.ID] = p
	return nil
}

func (r *mockProgramRepo) GetByID(ctx context.Context, id int64) (*livetv.Program, error) {
	p, ok := r.programs[id]
	if !ok {
		return nil, nil
	}
	return p, nil
}

func (r *mockProgramRepo) ListByChannel(ctx context.Context, channelID int64, from, to time.Time) ([]*livetv.Program, error) {
	return nil, nil
}

func (r *mockProgramRepo) ListByLibrary(ctx context.Context, libraryID int64, from, to time.Time) ([]*livetv.Program, error) {
	return nil, nil
}

func (r *mockProgramRepo) GetCurrentByChannel(ctx context.Context, channelID int64, now time.Time) (*livetv.Program, error) {
	return nil, nil
}

func (r *mockProgramRepo) DeleteByChannel(ctx context.Context, channelID int64) error {
	return nil
}

func (r *mockProgramRepo) DeleteByLibrary(ctx context.Context, libraryID int64) error {
	for id, p := range r.programs {
		if p.ChannelID == libraryID {
			delete(r.programs, id)
		}
	}
	return nil
}

func (r *mockProgramRepo) BulkCreate(ctx context.Context, programs []*livetv.Program) error {
	for _, p := range programs {
		p.ID = r.nextID
		r.nextID++
		r.programs[p.ID] = p
	}
	return nil
}

func TestFindM3UFiles(t *testing.T) {
	t.Run("finds .m3u files", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "channels.m3u"), []byte("#EXTM3U\n"), 0644)
		os.WriteFile(filepath.Join(dir, "other.txt"), []byte("hello"), 0644)

		files := findM3UFiles(dir)
		if len(files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(files))
		}
		if filepath.Base(files[0]) != "channels.m3u" {
			t.Errorf("expected channels.m3u, got %s", filepath.Base(files[0]))
		}
	})

	t.Run("falls back to .m3u8 when no .m3u found", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "playlist.m3u8"), []byte("#EXTM3U\n"), 0644)

		files := findM3UFiles(dir)
		if len(files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(files))
		}
		if filepath.Base(files[0]) != "playlist.m3u8" {
			t.Errorf("expected playlist.m3u8, got %s", filepath.Base(files[0]))
		}
	})

	t.Run("prefers .m3u over .m3u8", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "channels.m3u"), []byte("#EXTM3U\n"), 0644)
		os.WriteFile(filepath.Join(dir, "channels.m3u8"), []byte("#EXTM3U\n"), 0644)

		files := findM3UFiles(dir)
		if len(files) != 1 {
			t.Fatalf("expected 1 file (only .m3u), got %d", len(files))
		}
	})

	t.Run("returns nil for empty directory", func(t *testing.T) {
		dir := t.TempDir()
		files := findM3UFiles(dir)
		if files != nil {
			t.Errorf("expected nil, got %v", files)
		}
	})

	t.Run("returns nil for non-existent path", func(t *testing.T) {
		files := findM3UFiles("/nonexistent/path/that/does/not/exist")
		if files != nil {
			t.Errorf("expected nil, got %v", files)
		}
	})
}

func TestFindXMLTVFile(t *testing.T) {
	t.Run("finds guide.xml", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "guide.xml"), []byte("<?xml version='1.0'?>"), 0644)

		got := findXMLTVFile(dir)
		if got == "" {
			t.Fatal("expected non-empty path")
		}
		if filepath.Base(got) != "guide.xml" {
			t.Errorf("expected guide.xml, got %s", filepath.Base(got))
		}
	})

	t.Run("returns empty when no guide.xml exists", func(t *testing.T) {
		dir := t.TempDir()
		got := findXMLTVFile(dir)
		if got != "" {
			t.Errorf("expected empty string, got %s", got)
		}
	})

	t.Run("returns empty for non-existent path", func(t *testing.T) {
		got := findXMLTVFile("/nonexistent/path/that/does/not/exist")
		if got != "" {
			t.Errorf("expected empty string, got %s", got)
		}
	})
}

func TestImportEPG(t *testing.T) {
	xmltvContent := `<?xml version="1.0" encoding="UTF-8"?>
<tv>
  <channel id="BBC One">
    <display-name>BBC One</display-name>
  </channel>
  <programme start="20260613040200 +0000" stop="20260613050200 +0000" channel="BBC One">
    <title>News at Six</title>
    <desc>Evening news programme</desc>
    <category>News</category>
  </programme>
  <programme start="20260613050200 +0000" stop="20260613060200 +0000" channel="BBC One">
    <title>Movie Night</title>
    <category>Movie</category>
  </programme>
  <programme start="20260613060200 +0000" stop="20260613070200 +0000" channel="BBC Two">
    <title>Unmapped Show</title>
  </programme>
</tv>`

	t.Run("imports programs from XMLTV with EPGChannelID mapping", func(t *testing.T) {
		dir := t.TempDir()
		xmltvPath := filepath.Join(dir, "guide.xml")
		if err := os.WriteFile(xmltvPath, []byte(xmltvContent), 0644); err != nil {
			t.Fatal(err)
		}

		chRepo := newMockChannelRepo()
		chRepo.Upsert(context.Background(), &livetv.Channel{
			LibraryID:    1,
			Name:         "BBC One",
			EPGChannelID: "BBC One",
		})

		progRepo := newMockProgramRepo()
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))

		deps := &LiveTVDeps{
			ChannelRepo: chRepo,
			ProgramRepo: progRepo,
			EPGQuerier:  nil,
		}

		count, err := importEPG(context.Background(), deps, 1, xmltvPath, logger)
		if err != nil {
			t.Fatalf("importEPG() error = %v", err)
		}
		if count != 2 {
			t.Errorf("expected 2 programs, got %d", count)
		}
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		deps := &LiveTVDeps{
			ChannelRepo: newMockChannelRepo(),
			ProgramRepo: newMockProgramRepo(),
			EPGQuerier:  nil,
		}

		_, err := importEPG(context.Background(), deps, 1, "/nonexistent/guide.xml", logger)
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})

	t.Run("warns about unmapped XMLTV channels", func(t *testing.T) {
		dir := t.TempDir()
		xmltvPath := filepath.Join(dir, "guide.xml")
		if err := os.WriteFile(xmltvPath, []byte(xmltvContent), 0644); err != nil {
			t.Fatal(err)
		}

		chRepo := newMockChannelRepo()
		// No channel with EPGChannelID "BBC One" — it will use the fallback,
		// but "BBC Two" programs are unmapped
		chRepo.Upsert(context.Background(), &livetv.Channel{
			LibraryID:    1,
			Name:         "BBC One",
			EPGChannelID: "BBC One",
		})

		progRepo := newMockProgramRepo()
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))

		deps := &LiveTVDeps{
			ChannelRepo: chRepo,
			ProgramRepo: progRepo,
			EPGQuerier:  nil,
		}

		count, err := importEPG(context.Background(), deps, 1, xmltvPath, logger)
		if err != nil {
			t.Fatalf("importEPG() error = %v", err)
		}
		if count != 2 {
			t.Errorf("expected 2 mapped programs, got %d", count)
		}
	})
}
