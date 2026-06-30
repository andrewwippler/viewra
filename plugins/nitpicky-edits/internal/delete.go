package internal

import (
	"context"
	"database/sql"
	"fmt"
)

// deleteTVEpisode removes a TV episode from the database without deleting the media file.
func deleteTVEpisode(ctx context.Context, db *sql.DB, episodeID int64, driver string) error {
	// Delete watch progress for this episode
	if _, err := db.ExecContext(ctx, rebind("DELETE FROM watch_progress WHERE media_id = $1", driver), episodeID); err != nil {
		return fmt.Errorf("delete episode progress: %w", err)
	}
	// Delete the episode
	result, err := db.ExecContext(ctx, rebind("DELETE FROM tv_episodes WHERE id = $1", driver), episodeID)
	if err != nil {
		return fmt.Errorf("delete episode: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("episode not found: %d", episodeID)
	}
	return nil
}

// deleteTVSeason removes a TV season and its episodes from the database.
func deleteTVSeason(ctx context.Context, db *sql.DB, seasonID int64, driver string) error {
	// Delete watch progress for all episodes in this season
	if _, err := db.ExecContext(ctx,
		rebind("DELETE FROM watch_progress WHERE media_id IN (SELECT id FROM tv_episodes WHERE season_id = $1)", driver),
		seasonID,
	); err != nil {
		return fmt.Errorf("delete season progress: %w", err)
	}
	// Delete episodes
	if _, err := db.ExecContext(ctx, rebind("DELETE FROM tv_episodes WHERE season_id = $1", driver), seasonID); err != nil {
		return fmt.Errorf("delete season episodes: %w", err)
	}
	// Delete the season
	result, err := db.ExecContext(ctx, rebind("DELETE FROM tv_seasons WHERE id = $1", driver), seasonID)
	if err != nil {
		return fmt.Errorf("delete season: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("season not found: %d", seasonID)
	}
	return nil
}

// deleteTVShow removes a TV show and all its seasons and episodes from the database.
func deleteTVShow(ctx context.Context, db *sql.DB, showID int64, driver string) error {
	// Delete watch progress for all episodes in this show
	if _, err := db.ExecContext(ctx,
		rebind("DELETE FROM watch_progress WHERE media_id IN (SELECT id FROM tv_episodes WHERE show_id = $1)", driver),
		showID,
	); err != nil {
		return fmt.Errorf("delete show progress: %w", err)
	}
	// Delete episodes
	if _, err := db.ExecContext(ctx, rebind("DELETE FROM tv_episodes WHERE show_id = $1", driver), showID); err != nil {
		return fmt.Errorf("delete show episodes: %w", err)
	}
	// Delete seasons
	if _, err := db.ExecContext(ctx, rebind("DELETE FROM tv_seasons WHERE show_id = $1", driver), showID); err != nil {
		return fmt.Errorf("delete show seasons: %w", err)
	}
	// Delete ratings
	if _, err := db.ExecContext(ctx, rebind("DELETE FROM ratings WHERE entity_type = 'tv_show' AND entity_id = $1", driver), showID); err != nil {
		return fmt.Errorf("delete show ratings: %w", err)
	}
	// Delete the show
	result, err := db.ExecContext(ctx, rebind("DELETE FROM tv_shows WHERE id = $1", driver), showID)
	if err != nil {
		return fmt.Errorf("delete show: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("show not found: %d", showID)
	}
	return nil
}
