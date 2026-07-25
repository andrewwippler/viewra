package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/mantonx/viewra/internal/domain/media"
)

// PendingIdentificationRepository implements media.PendingIdentificationRepository using raw SQL.
type PendingIdentificationRepository struct {
	db     *sql.DB
	dbType string
}

// NewPendingIdentificationRepository creates a new pending identification repository.
func NewPendingIdentificationRepository(db *sql.DB, driver string) *PendingIdentificationRepository {
	return &PendingIdentificationRepository{
		db:     db,
		dbType: driver,
	}
}

// Insert adds a pending identification
func (r *PendingIdentificationRepository) Insert(ctx context.Context, id *media.PendingIdentification) error {
	now := time.Now()
	query := `INSERT INTO pending_identifications (library_id, file_path, provider, external_id, created_at)
		VALUES (?, ?, ?, ?, ?)`
	if r.dbType == "postgres" || r.dbType == "postgresql" {
		query = `INSERT INTO pending_identifications (library_id, file_path, provider, external_id, created_at)
			VALUES ($1, $2, $3, $4, $5) ON CONFLICT (file_path, provider) DO UPDATE SET external_id = $4`
	}

	result, err := r.db.ExecContext(ctx, query,
		id.LibraryID, id.FilePath, id.Provider, id.ExternalID, now)
	if err != nil {
		return fmt.Errorf("failed to insert pending identification: %w", err)
	}

	insertedID, _ := result.LastInsertId()
	id.ID = insertedID
	id.CreatedAt = now
	return nil
}

// GetByFilePath retrieves all pending IDs for a file path
func (r *PendingIdentificationRepository) GetByFilePath(ctx context.Context, filePath string) ([]*media.PendingIdentification, error) {
	query := `SELECT id, library_id, file_path, provider, external_id, created_at
		FROM pending_identifications WHERE file_path = ?`
	if r.dbType == "postgres" || r.dbType == "postgresql" {
		query = `SELECT id, library_id, file_path, provider, external_id, created_at
			FROM pending_identifications WHERE file_path = $1`
	}

	rows, err := r.db.QueryContext(ctx, query, filePath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// GetByLibrary retrieves all pending IDs for a library
func (r *PendingIdentificationRepository) GetByLibrary(ctx context.Context, libraryID int64) ([]*media.PendingIdentification, error) {
	query := `SELECT id, library_id, file_path, provider, external_id, created_at
		FROM pending_identifications WHERE library_id = ?`
	if r.dbType == "postgres" || r.dbType == "postgresql" {
		query = `SELECT id, library_id, file_path, provider, external_id, created_at
			FROM pending_identifications WHERE library_id = $1`
	}

	rows, err := r.db.QueryContext(ctx, query, libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// GetByLibraryAndPaths retrieves all pending IDs for multiple file paths in a library
func (r *PendingIdentificationRepository) GetByLibraryAndPaths(ctx context.Context, libraryID int64, filePaths []string) ([]*media.PendingIdentification, error) {
	if len(filePaths) == 0 {
		return []*media.PendingIdentification{}, nil
	}

	// Build query with placeholders
	query := `SELECT id, library_id, file_path, provider, external_id, created_at
		FROM pending_identifications WHERE library_id = ? AND file_path IN (`
	args := []interface{}{libraryID}

	for i, fp := range filePaths {
		if i > 0 {
			query += ", "
		}
		if r.dbType == "postgres" || r.dbType == "postgresql" {
			query += fmt.Sprintf("$%d", len(args)+1)
		} else {
			query += "?"
		}
		args = append(args, fp)
	}
	query += ")"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// Promote moves pending IDs to media_external_ids and deletes the pending records
func (r *PendingIdentificationRepository) Promote(ctx context.Context, filePath string, mediaID int64) error {
	// Get all pending IDs for this file
	pendingIDs, err := r.GetByFilePath(ctx, filePath)
	if err != nil {
		return err
	}

	if len(pendingIDs) == 0 {
		return nil
	}

	return r.PromoteBatch(ctx, filePath, mediaID, pendingIDs)
}

// PromoteBatch moves multiple pending IDs to media_external_ids
func (r *PendingIdentificationRepository) PromoteBatch(ctx context.Context, filePath string, mediaID int64, ids []*media.PendingIdentification) error {
	if len(ids) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert into media_external_ids
	for _, id := range ids {
		query := `INSERT INTO media_external_ids (media_id, provider, external_id, created_at)
			VALUES (?, ?, ?, ?) ON CONFLICT (media_id, provider) DO UPDATE SET external_id = ?`
		if r.dbType == "postgres" || r.dbType == "postgresql" {
			query = `INSERT INTO media_external_ids (media_id, provider, external_id, created_at)
				VALUES ($1, $2, $3, $4) ON CONFLICT (media_id, provider) DO UPDATE SET external_id = $5`
			_, err = tx.ExecContext(ctx, query, mediaID, id.Provider, id.ExternalID, id.CreatedAt, id.ExternalID)
		} else {
			_, err = tx.ExecContext(ctx, query, mediaID, id.Provider, id.ExternalID, id.CreatedAt, id.ExternalID)
		}
		if err != nil {
			return fmt.Errorf("failed to promote external ID: %w", err)
		}
	}

	// Delete from pending_identifications
	deleteQuery := `DELETE FROM pending_identifications WHERE file_path = ?`
	if r.dbType == "postgres" || r.dbType == "postgresql" {
		deleteQuery = `DELETE FROM pending_identifications WHERE file_path = $1`
	}
	_, err = tx.ExecContext(ctx, deleteQuery, filePath)
	if err != nil {
		return fmt.Errorf("failed to delete pending identifications: %w", err)
	}

	return tx.Commit()
}

// DeleteByFilePath removes all pending IDs for a file path
func (r *PendingIdentificationRepository) DeleteByFilePath(ctx context.Context, filePath string) error {
	query := `DELETE FROM pending_identifications WHERE file_path = ?`
	if r.dbType == "postgres" || r.dbType == "postgresql" {
		query = `DELETE FROM pending_identifications WHERE file_path = $1`
	}

	_, err := r.db.ExecContext(ctx, query, filePath)
	return err
}

// DeleteByLibrary removes all pending IDs for a library
func (r *PendingIdentificationRepository) DeleteByLibrary(ctx context.Context, libraryID int64) error {
	query := `DELETE FROM pending_identifications WHERE library_id = ?`
	if r.dbType == "postgres" || r.dbType == "postgresql" {
		query = `DELETE FROM pending_identifications WHERE library_id = $1`
	}

	_, err := r.db.ExecContext(ctx, query, libraryID)
	return err
}

// scanRows scans rows into a slice of PendingIdentification
func (r *PendingIdentificationRepository) scanRows(rows *sql.Rows) ([]*media.PendingIdentification, error) {
	var result []*media.PendingIdentification
	for rows.Next() {
		id := &media.PendingIdentification{}
		err := rows.Scan(&id.ID, &id.LibraryID, &id.FilePath, &id.Provider, &id.ExternalID, &id.CreatedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, rows.Err()
}
