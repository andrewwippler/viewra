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

// markSeasonWatched marks all episodes in a season as watched.
func markSeasonWatched(ctx context.Context, db *sql.DB, seasonID int64) error {
	rows, err := db.QueryContext(ctx, "SELECT id FROM tv_episodes WHERE season_id = ?", seasonID)
	if err != nil {
		return fmt.Errorf("mark season watched: query episodes: %w", err)
	}
	defer rows.Close()

	now := time.Now().Unix()
	for rows.Next() {
		var episodeID int64
		if err := rows.Scan(&episodeID); err != nil {
			return fmt.Errorf("mark season watched: scan episode: %w", err)
		}
		if _, err := db.ExecContext(ctx,
			`INSERT INTO watch_progress (media_id, is_watched, position, duration, updated_at)
			 VALUES (?, 1, 0, 0, ?)
			 ON CONFLICT(media_id) DO UPDATE SET is_watched = 1, position = 0, updated_at = ?`,
			episodeID, now, now,
		); err != nil {
			return fmt.Errorf("mark season watched: update episode %d: %w", episodeID, err)
		}
	}
	return rows.Err()
}

// markSeasonUnwatched marks all episodes in a season as unwatched.
func markSeasonUnwatched(ctx context.Context, db *sql.DB, seasonID int64) error {
	rows, err := db.QueryContext(ctx, "SELECT id FROM tv_episodes WHERE season_id = ?", seasonID)
	if err != nil {
		return fmt.Errorf("mark season unwatched: query episodes: %w", err)
	}
	defer rows.Close()

	now := time.Now().Unix()
	for rows.Next() {
		var episodeID int64
		if err := rows.Scan(&episodeID); err != nil {
			return fmt.Errorf("mark season unwatched: scan episode: %w", err)
		}
		if _, err := db.ExecContext(ctx,
			`INSERT INTO watch_progress (media_id, is_watched, position, duration, updated_at)
			 VALUES (?, 0, 0, 0, ?)
			 ON CONFLICT(media_id) DO UPDATE SET is_watched = 0, position = 0, updated_at = ?`,
			episodeID, now, now,
		); err != nil {
			return fmt.Errorf("mark season unwatched: update episode %d: %w", episodeID, err)
		}
	}
	return rows.Err()
}
