package livetv

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mantonx/viewra/internal/domain/library"
	"github.com/mantonx/viewra/internal/domain/livetv"
	infraLivetv "github.com/mantonx/viewra/internal/infrastructure/livetv"
)

type ScanChannelsUseCase struct {
	channelRepo livetv.ChannelRepository
	libraryRepo library.Repository
	httpClient  *http.Client
}

func NewScanChannelsUseCase(channelRepo livetv.ChannelRepository, libraryRepo library.Repository) *ScanChannelsUseCase {
	return &ScanChannelsUseCase{
		channelRepo: channelRepo,
		libraryRepo: libraryRepo,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (uc *ScanChannelsUseCase) Execute(ctx context.Context, libraryID int64) (ScanResponse, error) {
	lib, err := uc.libraryRepo.GetByID(ctx, libraryID)
	if err != nil {
		return ScanResponse{}, fmt.Errorf("failed to get library: %w", err)
	}

	if lib.Path == "" {
		return ScanResponse{}, fmt.Errorf("library has no M3U path configured")
	}

	var readers []io.Reader
	var sourceDescription string

	info, err := os.Stat(lib.Path)
	if err != nil {
		if os.IsNotExist(err) {
			// Path doesn't exist, try as URL
			return uc.fetchAndParseURL(ctx, lib.Path, lib)
		}
		return ScanResponse{}, fmt.Errorf("failed to stat path: %w", err)
	}

	if info.IsDir() {
		// Path is a directory - scan for .m3u files
		sourceDescription = fmt.Sprintf("directory %s", lib.Path)
		entries, err := filepath.Glob(filepath.Join(lib.Path, "*.m3u"))
		if err != nil {
			return ScanResponse{}, fmt.Errorf("failed to glob m3u files: %w", err)
		}
		if len(entries) == 0 {
			entries, err = filepath.Glob(filepath.Join(lib.Path, "*.m3u8"))
			if err != nil {
				return ScanResponse{}, fmt.Errorf("failed to glob m3u8 files: %w", err)
			}
		}
		for _, entry := range entries {
			f, err := os.Open(entry)
			if err != nil {
				return ScanResponse{}, fmt.Errorf("failed to open %s: %w", entry, err)
			}
			readers = append(readers, f)
			defer f.Close()
		}
	} else {
		// Path is a file - check extension
		ext := strings.ToLower(filepath.Ext(lib.Path))
		if ext == ".m3u" || ext == ".m3u8" {
			sourceDescription = fmt.Sprintf("file %s", lib.Path)
			f, err := os.Open(lib.Path)
			if err != nil {
				return ScanResponse{}, fmt.Errorf("failed to open %s: %w", lib.Path, err)
			}
			readers = append(readers, f)
			defer f.Close()
		} else {
			// Not a known playlist file, try as URL
			return uc.fetchAndParseURL(ctx, lib.Path, lib)
		}
	}

	if len(readers) == 0 {
		return ScanResponse{}, fmt.Errorf("no M3U files found in %s", lib.Path)
	}

	imported := 0
	for _, reader := range readers {
		entries, err := infraLivetv.ParseM3U(reader)
		if err != nil {
			return ScanResponse{}, fmt.Errorf("failed to parse M3U: %w", err)
		}

		entries, err = infraLivetv.CorrectSATIPPIDs(ctx, entries, lib, uc.libraryRepo, uc.channelRepo, uc.httpClient)
		if err != nil {
			return ScanResponse{}, fmt.Errorf("failed to correct SAT>IP PIDs: %w", err)
		}

		for _, entry := range entries {
			ch := entry.ToDomainChannel(libraryID)
			if err := uc.channelRepo.Upsert(ctx, ch); err != nil {
				return ScanResponse{}, fmt.Errorf("failed to upsert channel %q: %w", ch.Name, err)
			}
			imported++
		}
	}

	return ScanResponse{
		Message: fmt.Sprintf("Imported %d channels from %s", imported, sourceDescription),
	}, nil
}

func (uc *ScanChannelsUseCase) fetchAndParseURL(ctx context.Context, url string, lib *library.Library) (ScanResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ScanResponse{}, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := uc.httpClient.Do(req)
	if err != nil {
		return ScanResponse{}, fmt.Errorf("failed to fetch M3U: %w", err)
	}
	defer resp.Body.Close()

	entries, err := infraLivetv.ParseM3U(resp.Body)
	if err != nil {
		return ScanResponse{}, fmt.Errorf("failed to parse M3U: %w", err)
	}

	entries, err = infraLivetv.CorrectSATIPPIDs(ctx, entries, lib, uc.libraryRepo, uc.channelRepo, uc.httpClient)
	if err != nil {
		return ScanResponse{}, fmt.Errorf("failed to correct SAT>IP PIDs: %w", err)
	}

	imported := 0
	for _, entry := range entries {
		ch := entry.ToDomainChannel(lib.ID)
		if err := uc.channelRepo.Upsert(ctx, ch); err != nil {
			return ScanResponse{}, fmt.Errorf("failed to upsert channel %q: %w", ch.Name, err)
		}
		imported++
	}

	return ScanResponse{
		Message: fmt.Sprintf("Imported %d channels from URL", imported),
	}, nil
}
