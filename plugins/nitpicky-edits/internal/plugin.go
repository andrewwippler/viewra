package internal

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mantonx/viewra/pkg/plugin/sdk"
	"gopkg.in/yaml.v3"
)

var _ sdk.EnricherPlugin = (*Plugin)(nil)
var _ sdk.ConfigurableEnricher = (*Plugin)(nil)
var _ sdk.HTTPEnricher = (*Plugin)(nil)

type Plugin struct {
	sdk.Base
	logger *slog.Logger
	db     *sql.DB
	config *Config
}

func NewPlugin(logger *slog.Logger) *Plugin {
	return &Plugin{
		logger: logger,
		config: &Config{
			EditModeRequired: true,
		},
	}
}

func (p *Plugin) GetCapabilities() sdk.EnricherCapabilities {
	return sdk.EnricherCapabilities{
		MediaTypes: []string{"movie", "tv", "tv_show"},
		Provides:   []string{"external_ids"},
		IsLocal:    true,
	}
}

func (p *Plugin) Initialize(ctx context.Context, dataDir string, config []byte, services *sdk.HostServices) error {
	p.logger.Info("initializing nitpicky-edits plugin")

	if len(config) > 0 {
		var cfg Config
		if err := yaml.Unmarshal(config, &cfg); err != nil {
			if jsonErr := json.Unmarshal(config, &cfg); jsonErr != nil {
				return fmt.Errorf("failed to parse config: yaml: %w, json: %w", err, jsonErr)
			}
		}
		p.config = &cfg
	}

	if p.config.DBDriver == "" || p.config.DBDataSource == "" {
		p.logger.Warn("no database configuration provided - plugin will not be able to perform operations")
		return nil
	}

	db, err := openDB(p.config)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	p.db = db

	p.logger.Info("nitpicky-edits plugin initialized",
		"db_driver", p.config.DBDriver,
		"edit_mode_required", p.config.EditModeRequired,
	)
	return nil
}

func (p *Plugin) Shutdown(ctx context.Context) error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

func (p *Plugin) Enrich(ctx context.Context, req *sdk.EnrichRequest) (*sdk.EnrichResponse, error) {
	return &sdk.EnrichResponse{}, nil
}

func (p *Plugin) GetSettingsSchema() ([]byte, error) {
	return SettingsSchema()
}

func (p *Plugin) Configure(settings []byte) error {
	var cfg Config
	if err := json.Unmarshal(settings, &cfg); err != nil {
		return fmt.Errorf("failed to parse settings: %w", err)
	}
	if cfg.EditModeRequired != p.config.EditModeRequired {
		p.config.EditModeRequired = cfg.EditModeRequired
		p.logger.Info("edit_mode_required updated", "value", cfg.EditModeRequired)
	}
	return nil
}

func (p *Plugin) IsConfigured() bool {
	return p.db != nil
}

func (p *Plugin) GetRoutes() []sdk.Route {
	return []sdk.Route{
		{Path: "/identify/movie/:id", Methods: []string{"POST"}, AdminOnly: true, Description: "Set external IDs for a movie"},
		{Path: "/identify/tv/:id", Methods: []string{"POST"}, AdminOnly: true, Description: "Set external IDs for a TV show"},
		{Path: "/episode/:id", Methods: []string{"DELETE"}, AdminOnly: true, Description: "Delete a TV episode from the database"},
		{Path: "/season/:id", Methods: []string{"DELETE"}, AdminOnly: true, Description: "Delete a TV season and its episodes"},
		{Path: "/show/:id", Methods: []string{"DELETE"}, AdminOnly: true, Description: "Delete a TV show and all its seasons/episodes"},
		{Path: "/progress/:id/watched", Methods: []string{"POST"}, AdminOnly: true, Description: "Mark a media item as watched"},
		{Path: "/progress/:id/unwatched", Methods: []string{"POST"}, AdminOnly: true, Description: "Mark a media item as unwatched"},
		{Path: "/config", Methods: []string{"GET"}, AdminOnly: false, Description: "Get plugin configuration"},
	}
}

func (p *Plugin) HandleHTTP(ctx context.Context, req *sdk.HTTPRequest) (*sdk.HTTPResponse, error) {
	if p.db == nil {
		return sdk.JSONError(503, "plugin not initialized - database not configured")
	}

	path := strings.TrimPrefix(req.Path, "/")
	parts := strings.Split(path, "/")

	if len(parts) < 2 {
		return sdk.JSONError(404, "not found")
	}

	switch parts[0] {
	case "identify":
		return p.handleIdentify(ctx, req, parts)
	case "episode":
		return p.handleDeleteEpisode(ctx, parts)
	case "season":
		return p.handleDeleteSeason(ctx, parts)
	case "show":
		return p.handleDeleteShow(ctx, parts)
	case "progress":
		return p.handleProgress(ctx, parts)
	case "config":
		return p.handleGetConfig()
	default:
		return sdk.JSONError(404, "not found")
	}
}

func (p *Plugin) handleIdentify(ctx context.Context, req *sdk.HTTPRequest, parts []string) (*sdk.HTTPResponse, error) {
	if len(parts) < 3 {
		return sdk.JSONError(400, "invalid path: expected /identify/{type}/{id}")
	}

	mediaType := parts[1]
	id, err := parseInt64(parts[2])
	if err != nil {
		return sdk.JSONError(400, "invalid ID: "+err.Error())
	}

	switch mediaType {
	case "movie":
		var body IdentifyMovieRequest
		if err := json.Unmarshal(req.Body, &body); err != nil {
			return sdk.JSONError(400, "invalid request body: "+err.Error())
		}
		if err := identifyMovie(ctx, p.db, id, body); err != nil {
			return sdk.JSONError(500, err.Error())
		}
		return sdk.JSONResponse(200, map[string]interface{}{"status": "ok", "movie_id": id})

	case "tv":
		var body IdentifyTVShowRequest
		if err := json.Unmarshal(req.Body, &body); err != nil {
			return sdk.JSONError(400, "invalid request body: "+err.Error())
		}
		if err := identifyTVShow(ctx, p.db, id, body); err != nil {
			return sdk.JSONError(500, err.Error())
		}
		return sdk.JSONResponse(200, map[string]interface{}{"status": "ok", "show_id": id})

	default:
		return sdk.JSONError(400, fmt.Sprintf("unsupported media type: %s", mediaType))
	}
}

func (p *Plugin) handleDeleteEpisode(ctx context.Context, parts []string) (*sdk.HTTPResponse, error) {
	if len(parts) < 2 {
		return sdk.JSONError(400, "invalid path: expected /episode/{id}")
	}
	id, err := parseInt64(parts[1])
	if err != nil {
		return sdk.JSONError(400, "invalid ID: "+err.Error())
	}
	if err := deleteTVEpisode(ctx, p.db, id); err != nil {
		return sdk.JSONError(500, err.Error())
	}
	return sdk.EmptyResponse(204)
}

func (p *Plugin) handleDeleteSeason(ctx context.Context, parts []string) (*sdk.HTTPResponse, error) {
	if len(parts) < 2 {
		return sdk.JSONError(400, "invalid path: expected /season/{id}")
	}
	id, err := parseInt64(parts[1])
	if err != nil {
		return sdk.JSONError(400, "invalid ID: "+err.Error())
	}
	if err := deleteTVSeason(ctx, p.db, id); err != nil {
		return sdk.JSONError(500, err.Error())
	}
	return sdk.EmptyResponse(204)
}

func (p *Plugin) handleDeleteShow(ctx context.Context, parts []string) (*sdk.HTTPResponse, error) {
	if len(parts) < 2 {
		return sdk.JSONError(400, "invalid path: expected /show/{id}")
	}
	id, err := parseInt64(parts[1])
	if err != nil {
		return sdk.JSONError(400, "invalid ID: "+err.Error())
	}
	if err := deleteTVShow(ctx, p.db, id); err != nil {
		return sdk.JSONError(500, err.Error())
	}
	return sdk.EmptyResponse(204)
}

func (p *Plugin) handleProgress(ctx context.Context, parts []string) (*sdk.HTTPResponse, error) {
	if len(parts) < 3 {
		return sdk.JSONError(400, "invalid path: expected /progress/{id}/{action}")
	}

	id, err := parseInt64(parts[1])
	if err != nil {
		return sdk.JSONError(400, "invalid ID: "+err.Error())
	}

	switch parts[2] {
	case "watched":
		if err := markWatched(ctx, p.db, id); err != nil {
			return sdk.JSONError(500, err.Error())
		}
		return sdk.JSONResponse(200, map[string]interface{}{"status": "ok", "media_id": id, "is_watched": true})
	case "unwatched":
		if err := markUnwatched(ctx, p.db, id); err != nil {
			return sdk.JSONError(500, err.Error())
		}
		return sdk.JSONResponse(200, map[string]interface{}{"status": "ok", "media_id": id, "is_watched": false})
	default:
		return sdk.JSONError(400, fmt.Sprintf("unsupported progress action: %s", parts[2]))
	}
}

func (p *Plugin) handleGetConfig() (*sdk.HTTPResponse, error) {
	return sdk.JSONResponse(200, map[string]interface{}{
		"edit_mode_required": p.config.EditModeRequired,
	})
}

func parseInt64(s string) (int64, error) {
	var id int64
	for _, b := range []byte(s) {
		if b < '0' || b > '9' {
			return 0, fmt.Errorf("not a valid integer: %s", s)
		}
		id = id*10 + int64(b-'0')
	}
	return id, nil
}
