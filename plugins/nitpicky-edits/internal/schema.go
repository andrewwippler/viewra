package internal

import (
	"encoding/json"
	"fmt"
)

type Config struct {
	DBDriver         string `yaml:"db_driver" json:"db_driver"`
	DBDataSource     string `yaml:"db_data_source" json:"db_data_source"`
	EditModeRequired bool   `yaml:"edit_mode_required" json:"edit_mode_required"`
}

type IdentifyMovieRequest struct {
	IMDbID string `json:"imdb_id,omitempty"`
	TMDbID *int   `json:"tmdb_id,omitempty"`
}

type IdentifyTVShowRequest struct {
	IMDbID string `json:"imdb_id,omitempty"`
	TVDbID *int   `json:"tvdb_id,omitempty"`
	TMDbID *int   `json:"tmdb_id,omitempty"`
}

type IdentifyFileRequest struct {
	FilePath string `json:"file_path"`
	IMDbID   string `json:"imdb_id,omitempty"`
	TMDbID   *int   `json:"tmdb_id,omitempty"`
	TVDbID   *int   `json:"tvdb_id,omitempty"`
}

type MissingItem struct {
	MediaID       int64  `json:"media_id,omitempty"`
	ShowID        int64  `json:"show_id,omitempty"`
	ItemType      string `json:"item_type"`
	Title         string `json:"title"`
	FilePath      string `json:"file_path"`
	Year          *int   `json:"year,omitempty"`
	ShowTitle     string `json:"show_title,omitempty"`
	SeasonNumber  *int   `json:"season_number,omitempty"`
	EpisodeNumber *int   `json:"episode_number,omitempty"`
	LibraryName   string `json:"library_name,omitempty"`
}

type RemoveMissingRequest struct {
	MediaIDs []int64 `json:"media_ids"`
	ShowIDs  []int64 `json:"show_ids"`
}

func SettingsSchema() ([]byte, error) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"edit_mode_required": map[string]interface{}{
				"type":        "boolean",
				"title":       "Edit Mode Required",
				"description": "Require edit mode to be explicitly enabled before showing edit controls",
				"default":     true,
			},
		},
	}
	b, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("marshal settings schema: %w", err)
	}
	return b, nil
}
