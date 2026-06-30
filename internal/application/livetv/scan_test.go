package livetv

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/mantonx/viewra/internal/domain/library"
	"github.com/mantonx/viewra/internal/infrastructure/database/unified"
)

func TestScanChannelsUseCase_NoLibraryPath(t *testing.T) {
	libRepo := &mockLibraryRepo{
		lib: &library.Library{ID: 1, Path: ""},
	}
	uc := NewScanChannelsUseCase(&mockChannelRepo{}, libRepo)
	_, err := uc.Execute(context.Background(), 1)
	if err == nil {
		t.Error("expected error for empty library path")
	}
}

func TestScanChannelsUseCase_LibraryNotFound(t *testing.T) {
	libRepo := &mockLibraryRepo{getByIDErr: errors.New("not found")}
	uc := NewScanChannelsUseCase(&mockChannelRepo{}, libRepo)
	_, err := uc.Execute(context.Background(), 999)
	if err == nil {
		t.Error("expected error for missing library")
	}
}

func TestScanChannelsUseCase_FromDirectory(t *testing.T) {
	dir := t.TempDir()
	m3uContent := `#EXTM3U
#EXTINF:-1 tvg-id="bbc1" group-title="News",BBC One
http://stream.example.com/bbc1.m3u8
#EXTINF:-1 tvg-id="bbc2" group-title="News",BBC Two
http://stream.example.com/bbc2.m3u8
`
	if err := os.WriteFile(filepath.Join(dir, "channels.m3u"), []byte(m3uContent), 0644); err != nil {
		t.Fatal(err)
	}

	libRepo := &mockLibraryRepo{
		lib: &library.Library{ID: 1, Path: dir},
	}
	channelRepo := &mockChannelRepo{}
	uc := NewScanChannelsUseCase(channelRepo, libRepo)
	resp, err := uc.Execute(context.Background(), 1)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if resp.Message == "" {
		t.Error("expected non-empty message")
	}
	if len(channelRepo.upserted) != 2 {
		t.Fatalf("expected 2 upserted channels, got %d", len(channelRepo.upserted))
	}
	if channelRepo.upserted[0].Name != "BBC One" {
		t.Errorf("Name = %q", channelRepo.upserted[0].Name)
	}
	if channelRepo.upserted[1].Name != "BBC Two" {
		t.Errorf("Name = %q", channelRepo.upserted[1].Name)
	}
}

func TestScanChannelsUseCase_FromDirectoryNoM3UFiles(t *testing.T) {
	dir := t.TempDir()
	libRepo := &mockLibraryRepo{
		lib: &library.Library{ID: 1, Path: dir},
	}
	uc := NewScanChannelsUseCase(&mockChannelRepo{}, libRepo)
	_, err := uc.Execute(context.Background(), 1)
	if err == nil {
		t.Error("expected error for directory with no m3u files")
	}
}

func TestScanChannelsUseCase_FromSingleFile(t *testing.T) {
	m3uContent := `#EXTM3U
#EXTINF:-1 tvg-id="bbc1",BBC One
http://stream.example.com/bbc1.m3u8
`
	f, err := os.CreateTemp("", "*.m3u")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	if _, err := f.Write([]byte(m3uContent)); err != nil {
		t.Fatal(err)
	}
	f.Close()

	libRepo := &mockLibraryRepo{
		lib: &library.Library{ID: 1, Path: f.Name()},
	}
	channelRepo := &mockChannelRepo{}
	uc := NewScanChannelsUseCase(channelRepo, libRepo)
	resp, err := uc.Execute(context.Background(), 1)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if resp.Message == "" {
		t.Error("expected non-empty message")
	}
	if len(channelRepo.upserted) != 1 {
		t.Fatalf("expected 1 upserted channel, got %d", len(channelRepo.upserted))
	}
}

func TestScanChannelsUseCase_UpsertError(t *testing.T) {
	dir := t.TempDir()
	m3uContent := `#EXTM3U
#EXTINF:-1,Test
http://stream.example.com/test.m3u8
`
	if err := os.WriteFile(filepath.Join(dir, "test.m3u"), []byte(m3uContent), 0644); err != nil {
		t.Fatal(err)
	}
	libRepo := &mockLibraryRepo{
		lib: &library.Library{ID: 1, Path: dir},
	}
	channelRepo := &mockChannelRepo{upsertErr: errors.New("upsert failed")}
	uc := NewScanChannelsUseCase(channelRepo, libRepo)
	_, err := uc.Execute(context.Background(), 1)
	if err == nil {
		t.Error("expected error for upsert failure")
	}
}

func TestScanChannelsUseCase_FromDirectoryM3U8Extension(t *testing.T) {
	dir := t.TempDir()
	m3u8Content := `#EXTM3U
#EXTINF:-1,Test
http://stream.example.com/test.m3u8
`
	if err := os.WriteFile(filepath.Join(dir, "test.m3u8"), []byte(m3u8Content), 0644); err != nil {
		t.Fatal(err)
	}
	libRepo := &mockLibraryRepo{
		lib: &library.Library{ID: 1, Path: dir},
	}
	channelRepo := &mockChannelRepo{}
	uc := NewScanChannelsUseCase(channelRepo, libRepo)
	resp, err := uc.Execute(context.Background(), 1)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if resp.Message == "" {
		t.Error("expected non-empty message")
	}
	if len(channelRepo.upserted) != 1 {
		t.Fatalf("expected 1 upserted channel, got %d", len(channelRepo.upserted))
	}
}

func TestScanEPGUseCase_NoXMLTVPath(t *testing.T) {
	libRepo := &mockLibraryRepo{
		lib: &library.Library{ID: 1, Name: "TV", Type: "live_tv", Path: t.TempDir()},
	}
	uc := NewScanEPGUseCase(&mockChannelRepo{}, &mockProgramRepo{}, libRepo, unified.Querier{})
	_, err := uc.Execute(context.Background(), 1)
	if err == nil {
		t.Error("expected error when no XMLTV path configured")
	}
}

func TestScanEPGUseCase_NonExistentPath(t *testing.T) {
	libRepo := &mockLibraryRepo{
		lib: &library.Library{
			ID: 1, Name: "TV", Type: "live_tv",
			Path:             t.TempDir(),
			MonitoringConfig: &library.MonitoringConfig{XmltvURL: "/nonexistent/guide.xml"},
		},
	}
	uc := NewScanEPGUseCase(&mockChannelRepo{}, &mockProgramRepo{}, libRepo, unified.Querier{})
	_, err := uc.Execute(context.Background(), 1)
	if err == nil {
		t.Error("expected error for non-existent XMLTV path")
	}
}

func TestFindXMLTVFile(t *testing.T) {
	t.Run("finds guide.xml in directory", func(t *testing.T) {
		dir := t.TempDir()
		guidePath := filepath.Join(dir, "guide.xml")
		if err := os.WriteFile(guidePath, []byte("<tv/>"), 0644); err != nil {
			t.Fatal(err)
		}
		got := findXMLTVFile(dir)
		if got != guidePath {
			t.Errorf("got %q, want %q", got, guidePath)
		}
	})

	t.Run("returns empty when no guide.xml exists", func(t *testing.T) {
		dir := t.TempDir()
		got := findXMLTVFile(dir)
		if got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})
}
