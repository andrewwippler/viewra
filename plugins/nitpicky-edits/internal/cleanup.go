package internal

import (
	"context"
	"database/sql"
	"fmt"
	"os"
)

// scanMissingMedia checks all media files and TV show directories for existence
// on the filesystem and returns items whose files are missing.
func scanMissingMedia(ctx context.Context, db *sql.DB) ([]MissingItem, error) {
	items, err := scanMissingMediaFiles(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("scan media files: %w", err)
	}

	showItems, err := scanMissingShowDirectories(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("scan show directories: %w", err)
	}

	items = append(items, showItems...)
	return items, nil
}

// scanMissingMediaFiles checks all movie and TV episode file_paths for existence.
func scanMissingMediaFiles(ctx context.Context, db *sql.DB) ([]MissingItem, error) {
	query := `SELECT m.id, m.title, m.file_path, m.type,
		mo.year,
		te.season_number, te.episode_number,
		COALESCE(ts.title, '') AS show_title,
		COALESCE(l.name, '') AS library_name
	FROM media m
	LEFT JOIN movies mo ON mo.media_id = m.id
	LEFT JOIN tv_episodes te ON te.media_id = m.id
	LEFT JOIN tv_shows ts ON ts.id = te.show_id
	LEFT JOIN libraries l ON l.id = m.library_id
	WHERE m.type IN ('movie', 'tv_episode')
	ORDER BY m.type, m.title`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query media: %w", err)
	}
	defer rows.Close()

	var items []MissingItem
	for rows.Next() {
		var id int64
		var title, filePath, mediaType, showTitle, libName string
		var year, seasonNum, episodeNum sql.NullInt64

		if err := rows.Scan(&id, &title, &filePath, &mediaType,
			&year,
			&seasonNum, &episodeNum,
			&showTitle, &libName); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			item := MissingItem{
				MediaID:      id,
				ItemType:     mediaType,
				Title:        title,
				FilePath:     filePath,
				LibraryName:  libName,
				ShowTitle:    showTitle,
			}
			if year.Valid {
				y := int(year.Int64)
				item.Year = &y
			}
			if seasonNum.Valid {
				s := int(seasonNum.Int64)
				item.SeasonNumber = &s
			}
			if episodeNum.Valid {
				e := int(episodeNum.Int64)
				item.EpisodeNumber = &e
			}
			items = append(items, item)
		}
	}

	return items, rows.Err()
}

// scanMissingShowDirectories checks all TV show directories for existence.
func scanMissingShowDirectories(ctx context.Context, db *sql.DB) ([]MissingItem, error) {
	query := `SELECT id, title, COALESCE(directory, '') FROM tv_shows WHERE directory IS NOT NULL AND directory != ''`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query tv_shows: %w", err)
	}
	defer rows.Close()

	var items []MissingItem
	for rows.Next() {
		var id int64
		var title, directory string

		if err := rows.Scan(&id, &title, &directory); err != nil {
			return nil, fmt.Errorf("scan show row: %w", err)
		}

		if _, err := os.Stat(directory); os.IsNotExist(err) {
			items = append(items, MissingItem{
				ShowID:   id,
				ItemType: "tv_show",
				Title:    title,
				FilePath: directory,
			})
		}
	}

	return items, rows.Err()
}

// removeMissingItems deletes the specified media records and optionally entire
// TV shows from the database. It handles cascading cleanup of orphaned seasons
// and shows where all episodes have been removed.
func removeMissingItems(ctx context.Context, db *sql.DB, req RemoveMissingRequest, driver string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete individual media records (CASCADE handles movies, tv_episodes, watch_progress)
	if len(req.MediaIDs) > 0 {
		for _, id := range req.MediaIDs {
			if _, err := tx.ExecContext(ctx, rebind("DELETE FROM watch_progress WHERE media_id = $1", driver), id); err != nil {
				return fmt.Errorf("delete progress for media %d: %w", id, err)
			}
		}

		for _, id := range req.MediaIDs {
			if _, err := tx.ExecContext(ctx, rebind("DELETE FROM tv_episodes WHERE media_id = $1", driver), id); err != nil {
				return fmt.Errorf("delete tv_episode %d: %w", id, err)
			}
		}

		for _, id := range req.MediaIDs {
			if _, err := tx.ExecContext(ctx, rebind("DELETE FROM media WHERE id = $1", driver), id); err != nil {
				return fmt.Errorf("delete media %d: %w", id, err)
			}
		}

		// Clean up orphaned seasons (seasons with no remaining episodes)
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM tv_seasons WHERE id IN (
				SELECT s.id FROM tv_seasons s
				LEFT JOIN tv_episodes e ON e.season_id = s.id
		  WHERE e.media_id IS NULL
			)`,
		); err != nil {
			return fmt.Errorf("delete orphaned seasons: %w", err)
		}
	}

	// Delete entire TV shows
	if len(req.ShowIDs) > 0 {
		for _, id := range req.ShowIDs {
			if _, err := tx.ExecContext(ctx,
				rebind("DELETE FROM watch_progress WHERE media_id IN (SELECT media_id FROM tv_episodes WHERE show_id = $1)", driver),
				id,
			); err != nil {
				return fmt.Errorf("delete show %d progress: %w", id, err)
			}
			if _, err := tx.ExecContext(ctx, rebind("DELETE FROM tv_episodes WHERE show_id = $1", driver), id); err != nil {
				return fmt.Errorf("delete show %d episodes: %w", id, err)
			}
			if _, err := tx.ExecContext(ctx, rebind("DELETE FROM tv_seasons WHERE show_id = $1", driver), id); err != nil {
				return fmt.Errorf("delete show %d seasons: %w", id, err)
			}
			if _, err := tx.ExecContext(ctx, rebind("DELETE FROM user_ratings WHERE entity_type = 'tv_show' AND entity_id = $1", driver), id); err != nil {
				return fmt.Errorf("delete show %d ratings: %w", id, err)
			}
			result, err := tx.ExecContext(ctx, rebind("DELETE FROM tv_shows WHERE id = $1", driver), id)
			if err != nil {
				return fmt.Errorf("delete show %d: %w", id, err)
			}
			rows, _ := result.RowsAffected()
			if rows == 0 {
				return fmt.Errorf("show not found: %d", id)
			}
		}
	}

	// Clean up orphaned shows (shows with no remaining seasons)
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM tv_shows WHERE id IN (
			SELECT s.id FROM tv_shows s
			LEFT JOIN tv_seasons se ON se.show_id = s.id
			WHERE se.id IS NULL
		)`,
	); err != nil {
		return fmt.Errorf("delete orphaned shows: %w", err)
	}

	return tx.Commit()
}
