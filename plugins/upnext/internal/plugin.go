package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/mantonx/viewra/pkg/plugin/sdk"
)

type UpNextPlugin struct {
	sdk.Base
	mu      sync.RWMutex
	enabled bool
	data    *sdk.DataClient
}

var _ sdk.WidgetPlugin = (*UpNextPlugin)(nil)

func NewUpNextPlugin(logger *slog.Logger) *UpNextPlugin {
	p := &UpNextPlugin{enabled: true}
	p.SetLogger(logger)
	return p
}

func (p *UpNextPlugin) Initialize(ctx context.Context, dataDir string, config []byte, services *sdk.HostServices) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.Base.Init(dataDir)

	if services != nil {
		p.data = services.Data
	}

	p.Log().Debug("Up Next plugin initialized")
	return nil
}

func (p *UpNextPlugin) Shutdown(ctx context.Context) error {
	return nil
}

func (p *UpNextPlugin) GetSettingsSchema() ([]byte, error) {
	return SettingsSchema().Build()
}

func (p *UpNextPlugin) Configure(settings []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var cfg struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.Unmarshal(settings, &cfg); err != nil {
		return fmt.Errorf("failed to parse settings: %w", err)
	}
	p.enabled = cfg.Enabled
	return nil
}

func (p *UpNextPlugin) IsConfigured() bool {
	return true
}

func (p *UpNextPlugin) GetRoutes() []sdk.Route {
	return []sdk.Route{
		{
			Path:        "/up-next",
			Methods:     []string{"GET"},
			Description: "Get next unwatched TV episodes",
		},
	}
}

func (p *UpNextPlugin) HandleHTTP(ctx context.Context, req *sdk.HTTPRequest) (*sdk.HTTPResponse, error) {
	if req.Path == "/up-next" {
		return p.handleUpNext(ctx, req)
	}
	return jsonResponse(http.StatusNotFound, map[string]string{"error": "route not found"})
}

// handleUpNext returns the next unwatched episode for each TV show the user is watching.
// Uses a single efficient SQL query via the host's GetNextUnwatchedEpisodes RPC,
// replacing the previous N+1 query pattern with its string-based grouping and 50-episode limit.
func (p *UpNextPlugin) handleUpNext(ctx context.Context, req *sdk.HTTPRequest) (*sdk.HTTPResponse, error) {
	p.mu.RLock()
	enabled := p.enabled
	dataClient := p.data
	p.mu.RUnlock()

	if !enabled || dataClient == nil {
		return jsonResponse(http.StatusOK, map[string]any{
			"title": "Up Next",
			"items": []any{},
		})
	}

	episodes, err := dataClient.GetNextUnwatchedEpisodes(ctx, req.UserID, 10)
	if err != nil {
		p.Log().Warn("failed to get next unwatched episodes", "error", err)
		return jsonResponse(http.StatusOK, map[string]any{
			"title": "Up Next",
			"items": []any{},
		})
	}

	items := make([]any, 0, len(episodes))
	for _, ep := range episodes {
		items = append(items, map[string]any{
			"entity_type": "tv_show",
			"entity_id":   ep.ShowID,
			"title":       ep.ShowTitle,
			"progress": map[string]any{
				"percent":        0,
				"remaining_text": "Up next",
			},
			"episode_context": map[string]any{
				"season":           ep.SeasonNumber,
				"episode":          ep.EpisodeNumber,
				"episode_title":    ep.EpisodeTitle,
				"show_title":       ep.ShowTitle,
				"episode_media_id": ep.EpisodeMediaID,
			},
		})
	}

	return jsonResponse(http.StatusOK, map[string]any{
		"title": "Up Next",
		"items": items,
	})
}

func jsonResponse(statusCode int, data any) (*sdk.HTTPResponse, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return &sdk.HTTPResponse{
			StatusCode:  http.StatusInternalServerError,
			ContentType: "application/json",
			Body:        []byte(`{"error":"failed to serialize response"}`),
		}, nil
	}
	return &sdk.HTTPResponse{
		StatusCode:  statusCode,
		ContentType: "application/json",
		Body:        body,
	}, nil
}
