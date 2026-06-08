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
