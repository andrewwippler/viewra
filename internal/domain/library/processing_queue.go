package library

import (
	"context"
	"time"
)

// ProcessingItemStatus represents the status of a processing queue item
type ProcessingItemStatus string

const (
	ProcessingStatusPending    ProcessingItemStatus = "pending"
	ProcessingStatusProcessing ProcessingItemStatus = "processing"
	ProcessingStatusCompleted  ProcessingItemStatus = "completed"
	ProcessingStatusFailed     ProcessingItemStatus = "failed"
)

// ProcessingItem represents a file in the processing queue
type ProcessingItem struct {
	ID          int64
	LibraryID   int64
	FilePath    string
	FileHash    string
	Status      ProcessingItemStatus
	MediaID     *int64 // NULL until media record created
	ErrorMsg    string
	Attempts    int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ProcessingQueue defines the interface for processing queue operations
type ProcessingQueue interface {
	// Enqueue adds a file to the processing queue
	Enqueue(ctx context.Context, item *ProcessingItem) error

	// EnqueueBatch adds multiple files to the processing queue
	EnqueueBatch(ctx context.Context, items []*ProcessingItem) error

	// Dequeue claims the next pending item for processing
	// Returns nil, nil if queue is empty
	Dequeue(ctx context.Context) (*ProcessingItem, error)

	// Complete marks an item as completed with the created media ID
	Complete(ctx context.Context, id int64, mediaID int64) error

	// Fail marks an item as failed with an error message
	Fail(ctx context.Context, id int64, errMsg string) error

	// PendingCount returns the number of pending items for a library
	PendingCount(ctx context.Context, libraryID int64) (int64, error)

	// TotalPendingCount returns the total number of pending items across all libraries
	TotalPendingCount(ctx context.Context) (int64, error)

	// IsEmpty returns true if the queue has no pending items for a library
	IsEmpty(ctx context.Context, libraryID int64) (bool, error)

	// GetByFilePath retrieves a processing item by file path
	GetByFilePath(ctx context.Context, libraryID int64, filePath string) (*ProcessingItem, error)

	// ExistsInQueue checks if a file path is already in the queue
	ExistsInQueue(ctx context.Context, libraryID int64, filePath string) (bool, error)
}
