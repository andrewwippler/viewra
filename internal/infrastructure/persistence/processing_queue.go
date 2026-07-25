package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/mantonx/viewra/internal/domain/library"
)

// ProcessingQueueRepository implements library.ProcessingQueue using raw SQL.
type ProcessingQueueRepository struct {
	db     *sql.DB
	dbType string
}

// NewProcessingQueueRepository creates a new processing queue repository.
func NewProcessingQueueRepository(db *sql.DB, driver string) *ProcessingQueueRepository {
	return &ProcessingQueueRepository{
		db:     db,
		dbType: driver,
	}
}

// nowFunc is a function that returns the current time (for testing)
var nowFunc = time.Now

// Enqueue adds a file to the processing queue
func (r *ProcessingQueueRepository) Enqueue(ctx context.Context, item *library.ProcessingItem) error {
	now := nowFunc()
	query := `INSERT INTO processing_queue (library_id, file_path, file_hash, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`
	if r.dbType == "postgres" || r.dbType == "postgresql" {
		query = `INSERT INTO processing_queue (library_id, file_path, file_hash, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (library_id, file_path) DO NOTHING`
	}

	result, err := r.db.ExecContext(ctx, query,
		item.LibraryID, item.FilePath, item.FileHash,
		string(item.Status), now, now)
	if err != nil {
		return fmt.Errorf("failed to enqueue: %w", err)
	}

	id, _ := result.LastInsertId()
	item.ID = id
	item.CreatedAt = now
	item.UpdatedAt = now
	return nil
}

// EnqueueBatch adds multiple files to the processing queue
func (r *ProcessingQueueRepository) EnqueueBatch(ctx context.Context, items []*library.ProcessingItem) error {
	if len(items) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := nowFunc()
	for _, item := range items {
		query := `INSERT INTO processing_queue (library_id, file_path, file_hash, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)`
		if r.dbType == "postgres" || r.dbType == "postgresql" {
			query = `INSERT INTO processing_queue (library_id, file_path, file_hash, status, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (library_id, file_path) DO NOTHING`
		}

		result, err := tx.ExecContext(ctx, query,
			item.LibraryID, item.FilePath, item.FileHash,
			string(item.Status), now, now)
		if err != nil {
			return fmt.Errorf("failed to enqueue batch item: %w", err)
		}

		id, _ := result.LastInsertId()
		item.ID = id
		item.CreatedAt = now
		item.UpdatedAt = now
	}

	return tx.Commit()
}

// Dequeue claims the next pending item for processing
func (r *ProcessingQueueRepository) Dequeue(ctx context.Context) (*library.ProcessingItem, error) {
	now := nowFunc()

	// SQLite: Use a subquery to atomically claim the next item
	// PostgreSQL: Use FOR UPDATE SKIP LOCKED for better concurrency
	var row *sql.Row

	if r.dbType == "postgres" || r.dbType == "postgresql" {
		query := `UPDATE processing_queue
			SET status = 'processing', updated_at = $1, attempts = attempts + 1
			WHERE id = (
				SELECT id FROM processing_queue
				WHERE status = 'pending'
				ORDER BY created_at ASC
				LIMIT 1
				FOR UPDATE SKIP LOCKED
			)
			RETURNING id, library_id, file_path, file_hash, status, media_id,
				error_msg, attempts, created_at, updated_at`
		row = r.db.QueryRowContext(ctx, query, now)
	} else {
		// SQLite: Use a transaction to claim the next item
		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()

		// First, find the next pending item
		selectQuery := `SELECT id, library_id, file_path, file_hash, status, media_id,
			error_msg, attempts, created_at, updated_at
			FROM processing_queue WHERE status = 'pending'
			ORDER BY created_at ASC LIMIT 1`
		item := &library.ProcessingItem{}
		err = tx.QueryRowContext(ctx, selectQuery).Scan(
			&item.ID, &item.LibraryID, &item.FilePath, &item.FileHash,
			&item.Status, &item.MediaID, &item.ErrorMsg, &item.Attempts,
			&item.CreatedAt, &item.UpdatedAt)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, nil
			}
			return nil, err
		}

		// Then, update it to processing
		updateQuery := `UPDATE processing_queue SET status = 'processing',
			updated_at = ?, attempts = attempts + 1 WHERE id = ?`
		_, err = tx.ExecContext(ctx, updateQuery, now, item.ID)
		if err != nil {
			return nil, err
		}

		if err := tx.Commit(); err != nil {
			return nil, err
		}

		item.Status = library.ProcessingStatusProcessing
		item.UpdatedAt = now
		return item, nil
	}

	item := &library.ProcessingItem{}
	err := row.Scan(
		&item.ID, &item.LibraryID, &item.FilePath, &item.FileHash,
		&item.Status, &item.MediaID, &item.ErrorMsg, &item.Attempts,
		&item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return item, nil
}

// Complete marks an item as completed with the created media ID
func (r *ProcessingQueueRepository) Complete(ctx context.Context, id int64, mediaID int64) error {
	now := nowFunc()
	query := `UPDATE processing_queue SET status = 'completed', media_id = ?, updated_at = ? WHERE id = ?`
	if r.dbType == "postgres" || r.dbType == "postgresql" {
		query = `UPDATE processing_queue SET status = 'completed', media_id = $1, updated_at = $2 WHERE id = $3`
	}

	_, err := r.db.ExecContext(ctx, query, mediaID, now, id)
	return err
}

// Fail marks an item as failed with an error message
func (r *ProcessingQueueRepository) Fail(ctx context.Context, id int64, errMsg string) error {
	now := nowFunc()
	query := `UPDATE processing_queue SET status = 'failed', error_msg = ?, updated_at = ? WHERE id = ?`
	if r.dbType == "postgres" || r.dbType == "postgresql" {
		query = `UPDATE processing_queue SET status = 'failed', error_msg = $1, updated_at = $2 WHERE id = $3`
	}

	_, err := r.db.ExecContext(ctx, query, errMsg, now, id)
	return err
}

// PendingCount returns the number of pending items for a library
func (r *ProcessingQueueRepository) PendingCount(ctx context.Context, libraryID int64) (int64, error) {
	query := `SELECT COUNT(*) FROM processing_queue WHERE library_id = ? AND status IN ('pending', 'processing')`
	if r.dbType == "postgres" || r.dbType == "postgresql" {
		query = `SELECT COUNT(*) FROM processing_queue WHERE library_id = $1 AND status IN ('pending', 'processing')`
	}

	var count int64
	err := r.db.QueryRowContext(ctx, query, libraryID).Scan(&count)
	return count, err
}

// TotalPendingCount returns the total number of pending items across all libraries
func (r *ProcessingQueueRepository) TotalPendingCount(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM processing_queue WHERE status IN ('pending', 'processing')`

	var count int64
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}

// IsEmpty returns true if the queue has no pending items for a library
func (r *ProcessingQueueRepository) IsEmpty(ctx context.Context, libraryID int64) (bool, error) {
	count, err := r.PendingCount(ctx, libraryID)
	return count == 0, err
}

// GetByFilePath retrieves a processing item by file path
func (r *ProcessingQueueRepository) GetByFilePath(ctx context.Context, libraryID int64, filePath string) (*library.ProcessingItem, error) {
	query := `SELECT id, library_id, file_path, file_hash, status, media_id, error_msg, attempts,
		created_at, updated_at
		FROM processing_queue WHERE library_id = ? AND file_path = ?`
	if r.dbType == "postgres" || r.dbType == "postgresql" {
		query = `SELECT id, library_id, file_path, file_hash, status, media_id, error_msg, attempts,
			created_at, updated_at
			FROM processing_queue WHERE library_id = $1 AND file_path = $2`
	}

	item := &library.ProcessingItem{}
	err := r.db.QueryRowContext(ctx, query, libraryID, filePath).Scan(
		&item.ID, &item.LibraryID, &item.FilePath, &item.FileHash,
		&item.Status, &item.MediaID, &item.ErrorMsg, &item.Attempts,
		&item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

// ExistsInQueue checks if a file path is already in the queue
func (r *ProcessingQueueRepository) ExistsInQueue(ctx context.Context, libraryID int64, filePath string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM processing_queue WHERE library_id = ? AND file_path = ?)`
	if r.dbType == "postgres" || r.dbType == "postgresql" {
		query = `SELECT EXISTS(SELECT 1 FROM processing_queue WHERE library_id = $1 AND file_path = $2)`
	}

	var exists bool
	err := r.db.QueryRowContext(ctx, query, libraryID, filePath).Scan(&exists)
	return exists, err
}
