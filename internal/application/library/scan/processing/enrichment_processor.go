package processing

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mantonx/viewra/internal/application/library/scan/scanutil"
	"github.com/mantonx/viewra/internal/domain/library"
	"github.com/mantonx/viewra/internal/domain/scanner"
)

// EnrichmentFileProcessor implements FileProcessor for the enrichment worker.
// It processes files using FFprobe and creates media records.
type EnrichmentFileProcessor struct {
	deps           *Deps
	libraryRepo    library.Repository
	existingCache  *sync.Map
}

// NewEnrichmentFileProcessor creates a new file processor for the enrichment worker.
func NewEnrichmentFileProcessor(
	deps *Deps,
	libraryRepo library.Repository,
) *EnrichmentFileProcessor {
	return &EnrichmentFileProcessor{
		deps:          deps,
		libraryRepo:   libraryRepo,
		existingCache: &sync.Map{},
	}
}

// ProcessFile processes a file and returns the created media ID.
func (p *EnrichmentFileProcessor) ProcessFile(ctx context.Context, filePath string, libraryID int64) (int64, error) {
	// Get library info
	lib, err := p.libraryRepo.GetByID(ctx, libraryID)
	if err != nil {
		return 0, fmt.Errorf("failed to get library: %w", err)
	}

	// Stat the file with timeout
	fileInfo, err := scanutil.StatWithTimeout(ctx, filePath, p.deps.Config.BaseFileTimeout)
	if err != nil {
		return 0, fmt.Errorf("failed to stat file: %w", err)
	}

	// Create scanner.FileInfo
	scanFileInfo := scanner.FileInfo{
		Path:      filePath,
		Size:      fileInfo.Size(),
		ModTime:   fileInfo.ModTime(),
		IsDir:     false,
		Extension: strings.ToLower(filepath.Ext(filePath)),
	}

	// Calculate timeout based on file size
	timeout := calculateProcessingTimeout(p.deps, fileInfo.Size())
	processCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Process file with coordinator (FFprobe)
	result := p.deps.Coordinator.ProcessFile(processCtx, scanFileInfo)
	if result.Error != nil {
		if processCtx.Err() == context.DeadlineExceeded {
			return 0, fmt.Errorf("processing timeout after %v: %w", timeout, processCtx.Err())
		}
		return 0, result.Error
	}

	// Create checkpoint for media processing
	checkpoint := &scanner.ScanCheckpoint{
		FilePath: filePath,
		FileSize: fileInfo.Size(),
	}

	// Process based on library type
	var mediaID *int64
	var processErr error
	switch lib.Type {
	case library.LibraryTypeMovies:
		mediaID, processErr = p.deps.MediaProcessor.ProcessMovie(ctx, lib.ID, &result, checkpoint, p.existingCache)
	case library.LibraryTypeTV:
		mediaID, processErr = p.deps.MediaProcessor.ProcessTVEpisode(ctx, lib.ID, &result, checkpoint, p.existingCache)
	case library.LibraryTypeMusic:
		mediaID, processErr = p.deps.MediaProcessor.ProcessMusicTrack(ctx, lib.ID, &result, checkpoint, p.existingCache)
	default:
		return 0, fmt.Errorf("unknown library type: %s", lib.Type)
	}

	if processErr != nil {
		return 0, processErr
	}

	if mediaID == nil {
		return 0, fmt.Errorf("media ID is nil after processing")
	}

	return *mediaID, nil
}
