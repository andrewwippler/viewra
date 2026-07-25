package processing

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mantonx/viewra/internal/domain/library"
	"github.com/mantonx/viewra/internal/domain/media"
)

// EnrichmentWorker processes the processing queue in the background
type EnrichmentWorker struct {
	processingQueue library.ProcessingQueue
	pendingIDsRepo  media.PendingIdentificationRepository
	mediaRepo       media.Repository
	processor       FileProcessor
	logger          *slog.Logger
	pollInterval    time.Duration
}

// FileProcessor processes a single file (FFprobe, create media record, etc.)
type FileProcessor interface {
	// ProcessFile processes a file and returns the created media ID
	ProcessFile(ctx context.Context, filePath string, libraryID int64) (int64, error)
}

// NewEnrichmentWorker creates a new enrichment worker
func NewEnrichmentWorker(
	processingQueue library.ProcessingQueue,
	pendingIDsRepo media.PendingIdentificationRepository,
	mediaRepo media.Repository,
	processor FileProcessor,
	logger *slog.Logger,
) *EnrichmentWorker {
	if logger == nil {
		logger = slog.Default()
	}

	return &EnrichmentWorker{
		processingQueue: processingQueue,
		pendingIDsRepo:  pendingIDsRepo,
		mediaRepo:       mediaRepo,
		processor:       processor,
		logger:          logger,
		pollInterval:    5 * time.Second,
	}
}

// SetPollInterval sets how often to check for new items
func (w *EnrichmentWorker) SetPollInterval(interval time.Duration) {
	w.pollInterval = interval
}

// Run starts the worker loop
func (w *EnrichmentWorker) Run(ctx context.Context) {
	w.logger.Info("starting enrichment worker")

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("enrichment worker stopping")
			return
		default:
			// Try to dequeue an item
			item, err := w.processingQueue.Dequeue(ctx)
			if err != nil {
				w.logger.Error("failed to dequeue item", "error", err)
				time.Sleep(w.pollInterval)
				continue
			}

			if item == nil {
				// Queue empty, wait before checking again
				select {
				case <-ctx.Done():
					return
				case <-time.After(w.pollInterval):
				}
				continue
			}

			// Process the item
			w.processItem(ctx, item)
		}
	}
}

// processItem processes a single queue item
func (w *EnrichmentWorker) processItem(ctx context.Context, item *library.ProcessingItem) {
	w.logger.Info("processing item",
		"file_path", item.FilePath,
		"library_id", item.LibraryID,
		"attempts", item.Attempts)

	// Step 1: Process the file (FFprobe, create media record)
	mediaID, err := w.processor.ProcessFile(ctx, item.FilePath, item.LibraryID)
	if err != nil {
		w.logger.Error("failed to process file",
			"file_path", item.FilePath,
			"error", err)
		if failErr := w.processingQueue.Fail(ctx, item.ID, err.Error()); failErr != nil {
			w.logger.Error("failed to mark item as failed", "error", failErr)
		}
		return
	}

	// Step 2: Promote pending IDs if any
	if w.pendingIDsRepo != nil {
		if err := w.promotePendingIDs(ctx, item.FilePath, mediaID); err != nil {
			w.logger.Warn("failed to promote pending IDs",
				"file_path", item.FilePath,
				"error", err)
			// Continue - not fatal
		}
	}

	// Step 3: Mark as completed
	if err := w.processingQueue.Complete(ctx, item.ID, mediaID); err != nil {
		w.logger.Error("failed to mark item as completed",
			"item_id", item.ID,
			"error", err)
	}

	w.logger.Info("processed item successfully",
		"file_path", item.FilePath,
		"media_id", mediaID)
}

// promotePendingIDs moves pending IDs to media_external_ids
func (w *EnrichmentWorker) promotePendingIDs(ctx context.Context, filePath string, mediaID int64) error {
	// Get pending IDs for this file
	pendingIDs, err := w.pendingIDsRepo.GetByFilePath(ctx, filePath)
	if err != nil {
		return fmt.Errorf("failed to get pending IDs: %w", err)
	}

	if len(pendingIDs) == 0 {
		return nil
	}

	// Promote them
	return w.pendingIDsRepo.PromoteBatch(ctx, filePath, mediaID, pendingIDs)
}
