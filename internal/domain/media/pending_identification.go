package media

import (
	"context"
	"time"
)

// PendingIdentification represents an external ID set by a plugin before scan runs.
// During scan, these are promoted to media_external_ids.
type PendingIdentification struct {
	ID         int64
	LibraryID  int64
	FilePath   string
	Provider   string // "imdb", "tmdb", "tvdb", "musicbrainz"
	ExternalID string
	CreatedAt  time.Time
}

// PendingIdentificationRepository defines the interface for pending identification operations
type PendingIdentificationRepository interface {
	// Insert adds a pending identification
	Insert(ctx context.Context, id *PendingIdentification) error

	// GetByFilePath retrieves all pending IDs for a file path
	GetByFilePath(ctx context.Context, filePath string) ([]*PendingIdentification, error)

	// GetByLibrary retrieves all pending IDs for a library
	GetByLibrary(ctx context.Context, libraryID int64) ([]*PendingIdentification, error)

	// GetByLibraryAndPaths retrieves all pending IDs for multiple file paths in a library
	GetByLibraryAndPaths(ctx context.Context, libraryID int64, filePaths []string) ([]*PendingIdentification, error)

	// Promote moves pending IDs to media_external_ids and deletes the pending records
	Promote(ctx context.Context, filePath string, mediaID int64) error

	// PromoteBatch moves multiple pending IDs to media_external_ids
	PromoteBatch(ctx context.Context, filePath string, mediaID int64, ids []*PendingIdentification) error

	// DeleteByFilePath removes all pending IDs for a file path
	DeleteByFilePath(ctx context.Context, filePath string) error

	// DeleteByLibrary removes all pending IDs for a library
	DeleteByLibrary(ctx context.Context, libraryID int64) error
}
