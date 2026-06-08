package internal

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// markWatched marks a media item as fully watched.
func markWatched(ctx context.Context, db *sql.DB, mediaID int64) error {
	query := `INSERT INTO watch_progress (media_id, is_watched, position, duration, updated_at)
	           VALUES (?, 1, 0, 0, ?)
	           ON CONFLICT(media_id) DO UPDATE SET is_watched = 1, position = 0, updated_at = ?`
	now := time.Now().Unix()
	_, err := db.ExecContext(ctx, query, mediaID, now, now)
	if err != nil {
		return fmt.Errorf("mark watched: %w", err)
	}
	return nil
}

// markUnwatched marks a media item as unwatched.
func markUnwatched(ctx context.Context, db *sql.DB, mediaID int64) error {
	query := `INSERT INTO watch_progress (media_id, is_watched, position, duration, updated_at)
	           VALUES (?, 0, 0, 0, ?)
	           ON CONFLICT(media_id) DO UPDATE SET is_watched = 0, position = 0, updated_at = ?`
	now := time.Now().Unix()
	_, err := db.ExecContext(ctx, query, mediaID, now, now)
	if err != nil {
		return fmt.Errorf("mark unwatched: %w", err)
	}
	return nil
}
