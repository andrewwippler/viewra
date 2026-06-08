package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"sync"

	"github.com/mantonx/viewra/pkg/plugin/sdk"
)

type UpNextPlugin struct {
	sdk.Base
	mu       sync.RWMutex
	enabled  bool
	data     *sdk.DataClient
	progress *sdk.ProgressClient
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
		p.progress = services.Progress
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

func (p *UpNextPlugin) handleUpNext(ctx context.Context, req *sdk.HTTPRequest) (*sdk.HTTPResponse, error) {
	p.mu.RLock()
	enabled := p.enabled
	dataClient := p.data
	progressClient := p.progress
	p.mu.RUnlock()

	if !enabled || dataClient == nil || progressClient == nil {
		return jsonResponse(http.StatusOK, map[string]any{
			"title": "Up Next",
			"items": []any{},
		})
	}

	// Get watched TV episodes
	watched, err := progressClient.ListWatchedItems(ctx, req.UserID, "tv_episode", 100, 0)
	if err != nil {
		p.Log().Warn("failed to list watched items", "error", err)
		return jsonResponse(http.StatusOK, map[string]any{
			"title": "Up Next",
			"items": []any{},
		})
	}

	if len(watched) == 0 {
		return jsonResponse(http.StatusOK, map[string]any{
			"title": "Up Next",
			"items": []any{},
		})
	}

	p.Log().Debug("fetched watched episodes", "count", len(watched))

	// Get details for each watched episode to find show, season, episode
	type watchedEpisode struct {
		*sdk.WatchProgress
		episodeNumber int
		seasonNumber  int
		showTitle     string
		libraryID     int64
	}

	var watchedEps []watchedEpisode
	for _, w := range watched {
		details, err := dataClient.GetMediaDetails(ctx, w.MediaID, "tv_episode")
		if err != nil {
			continue
		}
		watchedEps = append(watchedEps, watchedEpisode{
			WatchProgress: w,
			episodeNumber: details.EpisodeNumber,
			seasonNumber:  details.SeasonNumber,
			showTitle:     details.ShowTitle,
			libraryID:     details.LibraryID,
		})
	}

	if len(watchedEps) == 0 {
		return jsonResponse(http.StatusOK, map[string]any{
			"title": "Up Next",
			"items": []any{},
		})
	}

	// Group by show title, keep latest watched episode per show
	showLatest := make(map[string]*watchedEpisode)
	for i := range watchedEps {
		ep := &watchedEps[i]
		existing, ok := showLatest[ep.showTitle]
		if !ok || ep.LastWatchedAt.After(existing.LastWatchedAt) {
			showLatest[ep.showTitle] = ep
		}
	}

	// Sort shows by most recently watched
	type showEntry struct {
		title string
		ep    *watchedEpisode
	}
	var sortedShows []showEntry
	for title, ep := range showLatest {
		sortedShows = append(sortedShows, showEntry{title, ep})
	}
	sort.Slice(sortedShows, func(i, j int) bool {
		return sortedShows[i].ep.LastWatchedAt.After(sortedShows[j].ep.LastWatchedAt)
	})

	// For each show, search for episodes to find the next one
	type nextEpisode struct {
		entityType    string
		entityID      int64 // show ID (for navigation and backdrop images)
		episodeID     int64 // episode media ID (for playback)
		title         string
		episodeTitle  string
		seasonNumber  int
		episodeNumber int
		showTitle     string
		libraryID     int64
	}

	var nextEps []nextEpisode

	for _, show := range sortedShows {
		if len(nextEps) >= 10 {
			break
		}

		ep := show.ep

		// Search for the show to get its ID (for navigation and backdrop images)
		showResults, showErr := dataClient.SearchMedia(ctx, ep.showTitle, 0, "tv", 1)
		var showID int64
		if showErr == nil && len(showResults) > 0 {
			showID = showResults[0].ID
		}
		if showID == 0 {
			p.Log().Debug("show not found for up next", "show", ep.showTitle)
			continue
		}

		// Build set of watched (season, episode) pairs for this show
		watchedSet := make(map[[2]int]bool)
		for _, we := range watchedEps {
			if we.showTitle == ep.showTitle {
				watchedSet[[2]int{we.seasonNumber, we.episodeNumber}] = true
			}
		}

		// Search for all episodes of this show
		searchResults, err := dataClient.SearchMedia(ctx, ep.showTitle, 0, "tv_episode", 50)
		if err != nil {
			p.Log().Debug("search failed for show", "show", ep.showTitle, "error", err)
			continue
		}

		if len(searchResults) == 0 {
			continue
		}

		// Get details for each result to find season/episode
		type epInfo struct {
			season  int
			episode int
			mediaID int64
			title   string
		}
		var allEps []epInfo
		for _, m := range searchResults {
			details, err := dataClient.GetMediaDetails(ctx, m.ID, "tv_episode")
			if err != nil {
				continue
			}
			allEps = append(allEps, epInfo{
				season:  details.SeasonNumber,
				episode: details.EpisodeNumber,
				mediaID: details.ID,
				title:   details.Title,
			})
		}

		if len(allEps) == 0 {
			continue
		}

		// Sort by (season, episode)
		sort.Slice(allEps, func(i, j int) bool {
			if allEps[i].season != allEps[j].season {
				return allEps[i].season < allEps[j].season
			}
			return allEps[i].episode < allEps[j].episode
		})

		// Find first unwatched episode after the latest watched one
		for _, e := range allEps {
			key := [2]int{e.season, e.episode}
			if !watchedSet[key] {
				// Found the next unwatched episode
				// Only include if it's strictly after the last watched episode
				// Not the same episode or before
				if e.season > ep.seasonNumber ||
					(e.season == ep.seasonNumber && e.episode > ep.episodeNumber) {
					nextEps = append(nextEps, nextEpisode{
						entityType:    "tv_show",
						entityID:      showID,
						episodeID:     e.mediaID,
						title:         ep.showTitle,
						episodeTitle:  e.title,
						seasonNumber:  e.season,
						episodeNumber: e.episode,
						showTitle:     ep.showTitle,
						libraryID:     ep.libraryID,
					})
					break
				}
			}
		}
	}

	// Convert to continue-watching items (wide 16:9 cards)
	items := make([]any, 0, len(nextEps))

	for _, ne := range nextEps {
		items = append(items, map[string]any{
			"entity_type": "tv_show",
			"entity_id":   ne.entityID,
			"title":       ne.title,
			"progress": map[string]any{
				"percent":        0,
				"remaining_text": "Up next",
			},
			"episode_context": map[string]any{
				"season":           ne.seasonNumber,
				"episode":          ne.episodeNumber,
				"episode_title":    ne.episodeTitle,
				"show_title":       ne.showTitle,
				"episode_media_id": ne.episodeID,
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
