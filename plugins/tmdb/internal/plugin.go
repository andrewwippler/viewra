// Package internal implements the TMDb plugin logic.
package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"sync"

	"github.com/mantonx/viewra/pkg/plugin/sdk"
	"gopkg.in/yaml.v3"
)

// Config holds the TMDb plugin configuration.
type Config struct {
	APIKey        string `yaml:"api_key" json:"api_key"`
	RateLimit     int    `yaml:"rate_limit" json:"rate_limit"`           // requests per 10 seconds (default: 40)
	CacheTTLHours int    `yaml:"cache_ttl_hours" json:"cache_ttl_hours"` // cache duration (default: 24)
	Language      string `yaml:"language" json:"language"`               // preferred language (default: en-US)

	// Settings from UI
	IncludeAdult     bool     `yaml:"include_adult" json:"include_adult"`
	FetchImages      bool     `yaml:"fetch_images" json:"fetch_images"`
	ImageTypes       []string `yaml:"image_types" json:"image_types"`
	ImageSize        string   `yaml:"image_size" json:"image_size"`
	FetchActorPhotos bool     `yaml:"fetch_actor_photos" json:"fetch_actor_photos"`
	MaxActorPhotos   int      `yaml:"max_actor_photos" json:"max_actor_photos"`
}

// TMDbPlugin implements sdk.EnricherPlugin for TMDb.
type TMDbPlugin struct {
	sdk.Base

	logger  *slog.Logger
	dataDir string
	config  Config
	client  *Client
	storage *sdk.StorageClient

	mu sync.RWMutex

	// Stats for health reporting
	requestsTotal int64
	errorsTotal   int64
}

// NewTMDbPlugin creates a new TMDb plugin instance.
func NewTMDbPlugin(logger *slog.Logger) *TMDbPlugin {
	p := &TMDbPlugin{
		logger: logger,
	}
	p.SetLogger(logger)
	return p
}

// recordError increments the error counter (thread-safe).
func (p *TMDbPlugin) recordError() {
	p.mu.Lock()
	p.errorsTotal++
	p.mu.Unlock()
}

// --- sdk.EnricherPlugin implementation ---

func (p *TMDbPlugin) GetCapabilities() sdk.EnricherCapabilities {
	return sdk.EnricherCapabilities{
		MediaTypes: []string{"movie", "tv", "tv_show"},
		Provides:   []string{"metadata", "artwork", "external_ids"},
		IsLocal:    false,
		RateLimit:  40, // TMDb allows ~40 requests per 10 seconds
		Requires:   []string{},
		Priority:   50,
	}
}

func (p *TMDbPlugin) Initialize(ctx context.Context, dataDir string, config []byte, services *sdk.HostServices) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.dataDir = dataDir
	p.logger.Debug("initializing TMDb plugin", "data_dir", dataDir)

	// Store host storage client if available
	if services != nil && services.Storage != nil {
		p.storage = services.Storage
		p.logger.Debug("host storage service available")
	}

	// Parse config from YAML
	if len(config) == 0 {
		return fmt.Errorf("config.yml is required but was not provided")
	}

	if err := yaml.Unmarshal(config, &p.config); err != nil {
		return fmt.Errorf("failed to parse config.yml: %w", err)
	}

	// Validate required fields
	if p.config.APIKey == "" {
		return fmt.Errorf("api_key is required in config.yml")
	}

	// Apply defaults
	if p.config.RateLimit == 0 {
		p.config.RateLimit = 40
	}
	if p.config.CacheTTLHours == 0 {
		p.config.CacheTTLHours = 24
	}
	if p.config.Language == "" {
		p.config.Language = "en-US"
	}

	// Create the API client
	client, err := NewClient(ClientConfig{
		APIKey:        p.config.APIKey,
		CacheTTLHours: p.config.CacheTTLHours,
		Storage:       p.storage,
		Logger:        p.logger,
	})
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}
	p.client = client

	cacheStatus := "disabled (no host storage)"
	if p.storage != nil {
		cacheStatus = fmt.Sprintf("enabled (TTL: %dh)", p.config.CacheTTLHours)
	}
	p.logger.Debug("TMDb plugin initialized",
		"rate_limit", p.config.RateLimit,
		"cache", cacheStatus,
		"language", p.config.Language)

	return nil
}

func (p *TMDbPlugin) Shutdown(ctx context.Context) error {
	p.logger.Debug("shutting down TMDb plugin")
	if p.client != nil {
		p.client.Close()
	}
	return nil
}

// GetSettingsSchema returns the JSON Schema for plugin settings.
func (p *TMDbPlugin) GetSettingsSchema() ([]byte, error) {
	return SettingsSchema().Build()
}

// Configure applies new settings to the plugin.
func (p *TMDbPlugin) Configure(settings []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var newSettings struct {
		Language         string   `json:"language"`
		IncludeAdult     bool     `json:"include_adult"`
		CacheTTLHours    int      `json:"cache_ttl_hours"`
		FetchImages      bool     `json:"fetch_images"`
		ImageTypes       []string `json:"image_types"`
		ImageSize        string   `json:"image_size"`
		FetchActorPhotos bool     `json:"fetch_actor_photos"`
		MaxActorPhotos   int      `json:"max_actor_photos"`
	}
	if err := json.Unmarshal(settings, &newSettings); err != nil {
		return fmt.Errorf("failed to parse settings: %w", err)
	}

	// Apply settings to config
	if newSettings.Language != "" {
		p.config.Language = newSettings.Language
	}
	p.config.IncludeAdult = newSettings.IncludeAdult
	if newSettings.CacheTTLHours > 0 {
		p.config.CacheTTLHours = newSettings.CacheTTLHours
	}
	p.config.FetchImages = newSettings.FetchImages
	if len(newSettings.ImageTypes) > 0 {
		p.config.ImageTypes = newSettings.ImageTypes
	}
	if newSettings.ImageSize != "" {
		p.config.ImageSize = newSettings.ImageSize
	}
	p.config.FetchActorPhotos = newSettings.FetchActorPhotos
	if newSettings.MaxActorPhotos > 0 {
		p.config.MaxActorPhotos = newSettings.MaxActorPhotos
	}

	p.logger.Debug("configuration updated",
		"language", p.config.Language,
		"include_adult", p.config.IncludeAdult,
		"cache_ttl_hours", p.config.CacheTTLHours,
		"fetch_images", p.config.FetchImages,
		"image_types", p.config.ImageTypes,
		"fetch_actor_photos", p.config.FetchActorPhotos,
	)

	return nil
}

func (p *TMDbPlugin) IsConfigured() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.config.APIKey != ""
}

func (p *TMDbPlugin) Enrich(ctx context.Context, req *sdk.EnrichRequest) (*sdk.EnrichResponse, error) {
	p.mu.Lock()
	p.requestsTotal++
	client := p.client
	p.mu.Unlock()

	// Check if configured
	if client == nil {
		p.mu.Lock()
		p.errorsTotal++
		p.mu.Unlock()
		return sdk.Skip("TMDb API key not configured"), nil
	}

	p.logger.Debug("enriching media",
		"media_id", req.MediaID,
		"media_type", req.MediaType,
		"title", req.Title,
		"year", req.Year,
	)

	switch req.MediaType {
	case "movie":
		return p.enrichMovie(ctx, client, req)
	case "tv", "tv_show":
		return p.enrichTV(ctx, client, req)
	default:
		p.logger.Debug("skipping TMDb enrichment for unsupported media type", "media_type", req.MediaType)
		return sdk.Skip("unsupported media type: " + req.MediaType), nil
	}
}

// --- sdk.HTTPEnricher implementation ---

func (p *TMDbPlugin) GetRoutes() []sdk.Route {
	return []sdk.Route{
		{
			Path:        "/enrich",
			Methods:     []string{"POST"},
			AdminOnly:   true,
			Description: "Trigger enrichment for a specific media item (for testing)",
		},
		{
			Path:        "/lookup",
			Methods:     []string{"GET"},
			AdminOnly:   false,
			Description: "Look up a title on TMDB without enriching",
		},
		{
			Path:        "/trending",
			Methods:     []string{"GET"},
			AdminOnly:   false,
			Description: "Get trending movies and TV shows from TMDb",
		},
		{
			Path:        "/trending/info",
			Methods:     []string{"GET"},
			AdminOnly:   false,
			Description: "Get trending provider metadata",
		},
	}
}

func (p *TMDbPlugin) HandleHTTP(ctx context.Context, req *sdk.HTTPRequest) (*sdk.HTTPResponse, error) {
	p.logger.Debug("handling HTTP request", "path", req.Path, "method", req.Method)

	switch req.Path {
	case "/enrich":
		return p.handleEnrich(ctx, req)
	case "/lookup":
		return p.handleLookup(ctx, req)
	case "/trending":
		return p.handleTrending(ctx, req)
	case "/trending/info":
		return p.handleTrendingInfo(ctx, req)
	default:
		return sdk.JSONError(http.StatusNotFound, "route not found: "+req.Path)
	}
}

func (p *TMDbPlugin) handleEnrich(ctx context.Context, req *sdk.HTTPRequest) (*sdk.HTTPResponse, error) {
	if req.Method != "POST" {
		return sdk.JSONError(http.StatusMethodNotAllowed, "method not allowed")
	}

	var enrichReq struct {
		MediaID     int64             `json:"media_id"`
		MediaType   string            `json:"media_type"`
		Title       string            `json:"title"`
		Year        int               `json:"year"`
		ExternalIDs map[string]string `json:"external_ids,omitempty"`
	}
	if err := json.Unmarshal(req.Body, &enrichReq); err != nil {
		return sdk.JSONError(http.StatusBadRequest, "invalid JSON: "+err.Error())
	}

	if enrichReq.MediaType == "" {
		return sdk.JSONError(http.StatusBadRequest, "media_type is required")
	}
	if enrichReq.Title == "" && len(enrichReq.ExternalIDs) == 0 {
		return sdk.JSONError(http.StatusBadRequest, "title or external_ids is required")
	}

	// Build the SDK enrich request
	sdkReq := &sdk.EnrichRequest{
		MediaID:     enrichReq.MediaID,
		MediaType:   enrichReq.MediaType,
		Title:       enrichReq.Title,
		Year:        enrichReq.Year,
		ExistingIDs: enrichReq.ExternalIDs,
	}

	// Call the enrichment logic
	resp, err := p.Enrich(ctx, sdkReq)
	if err != nil {
		return sdk.JSONError(http.StatusInternalServerError, err.Error())
	}

	// Build response with metadata summary
	result := map[string]any{
		"success": !resp.Skipped,
	}
	if resp.Skipped {
		result["skip_reason"] = resp.SkipReason
	}
	if resp.Metadata != nil {
		metaMap := map[string]any{
			"genres":     resp.Metadata.Genres,
			"cast_count": len(resp.Metadata.Cast),
		}
		if resp.Metadata.Title != nil {
			metaMap["title"] = *resp.Metadata.Title
		}
		if resp.Metadata.Year != nil {
			metaMap["year"] = *resp.Metadata.Year
		}
		if resp.Metadata.Plot != nil {
			metaMap["plot"] = truncate(*resp.Metadata.Plot, 200)
		}
		result["metadata"] = metaMap

		// Show keyword details if present
		if len(resp.Metadata.Keywords) > 0 {
			keywords := make([]map[string]any, 0, len(resp.Metadata.Keywords))
			locationCount := 0
			for _, kw := range resp.Metadata.Keywords {
				keywords = append(keywords, map[string]any{
					"id":          kw.ID,
					"name":        kw.Name,
					"is_location": kw.IsLocation,
				})
				if kw.IsLocation {
					locationCount++
				}
			}
			result["keywords"] = keywords
			result["location_keywords_count"] = locationCount
		}
	}

	return sdk.JSONResponse(http.StatusOK, result)
}

// imdbIDRegex matches IMDb ID format (tt followed by digits)
var imdbIDRegex = regexp.MustCompile(`^tt\d+$`)

// detectInputType determines if the input is an IMDb ID or a title search.
func detectInputType(input string) string {
	if imdbIDRegex.MatchString(input) {
		return "IMDB_LOOKUP"
	}
	return "TITLE_SEARCH"
}

// handleLookup handles the /lookup endpoint with dual-input logic:
// - IMDb ID (e.g., "tt0111161") -> uses /find endpoint for exact match
// - Title (e.g., "The Shawshank Redemption") -> uses search, enriches top results with IMDb IDs
func (p *TMDbPlugin) handleLookup(ctx context.Context, req *sdk.HTTPRequest) (*sdk.HTTPResponse, error) {
	if req.Method != "GET" {
		return sdk.JSONError(http.StatusMethodNotAllowed, "method not allowed")
	}

	query := req.Query["title"]
	mediaType := req.Query["type"]
	if mediaType == "" {
		mediaType = "movie"
	}

	if query == "" {
		return sdk.JSONError(http.StatusBadRequest, "title query param is required")
	}

	p.mu.RLock()
	client := p.client
	p.mu.RUnlock()

	if client == nil {
		return sdk.JSONError(http.StatusServiceUnavailable, "TMDb client not initialized")
	}

	inputType := detectInputType(query)

	response := LookupResponse{
		InputType: inputType,
		Query:     query,
		Type:      mediaType,
	}

	switch inputType {
	case "IMDB_LOOKUP":
		return p.handleIMDBLookup(ctx, client, mediaType, query, &response)
	case "TITLE_SEARCH":
		return p.handleTitleSearch(ctx, client, mediaType, query, &response)
	default:
		return sdk.JSONError(http.StatusBadRequest, "invalid input type")
	}
}

// handleIMDBLookup uses TMDb's /find endpoint to get exact match by IMDb ID.
func (p *TMDbPlugin) handleIMDBLookup(ctx context.Context, client *Client, mediaType, imdbID string, response *LookupResponse) (*sdk.HTTPResponse, error) {
	findResp, err := client.FindByIMDbID(ctx, imdbID)
	if err != nil {
		p.recordError()
		return sdk.JSONError(http.StatusInternalServerError, fmt.Sprintf("IMDb lookup failed: %v", err))
	}

	// Check movie results first
	if mediaType == "movie" || mediaType == "all" {
		if len(findResp.MovieResults) > 0 {
			movie := findResp.MovieResults[0]
			// Fetch full details to get images, etc.
			details, err := client.GetMovieDetails(ctx, movie.ID)
			if err != nil {
				p.logger.Warn("failed to fetch movie details", "tmdb_id", movie.ID, "error", err)
				// Fall back to search result data
				response.SingleResult = &EnrichedMovieSearchResult{
					MovieSearchResult: movie,
					IMDbID:            imdbID,
				}
			} else {
				response.SingleResult = &EnrichedMovieSearchResult{
					MovieSearchResult: MovieSearchResult{
						ID:               details.ID,
						Title:            details.Title,
						OriginalTitle:    details.OriginalTitle,
						Overview:         details.Overview,
						ReleaseDate:      details.ReleaseDate,
						PosterPath:       details.PosterPath,
						BackdropPath:     details.BackdropPath,
						VoteAverage:      details.VoteAverage,
						VoteCount:        details.VoteCount,
						OriginalLanguage: details.OriginalLanguage,
					},
					IMDbID: imdbID,
				}
			}
			return sdk.JSONResponse(http.StatusOK, response)
		}
	}

	// Check TV results
	if mediaType == "tv" || mediaType == "tv_show" || mediaType == "all" {
		if len(findResp.TVResults) > 0 {
			tv := findResp.TVResults[0]
			details, err := client.GetTVDetails(ctx, tv.ID)
			if err != nil {
				p.logger.Warn("failed to fetch TV details", "tmdb_id", tv.ID, "error", err)
				response.SingleTVResult = &EnrichedTVSearchResult{
					TVSearchResult: tv,
					IMDbID:         imdbID,
				}
			} else {
				response.SingleTVResult = &EnrichedTVSearchResult{
					TVSearchResult: TVSearchResult{
						ID:               details.ID,
						Name:             details.Name,
						OriginalName:     details.OriginalName,
						Overview:         details.Overview,
						FirstAirDate:     details.FirstAirDate,
						PosterPath:       details.PosterPath,
						BackdropPath:     details.BackdropPath,
						VoteAverage:      details.VoteAverage,
						VoteCount:        details.VoteCount,
						OriginalLanguage: details.OriginalLanguage,
					},
					IMDbID: imdbID,
				}
			}
			return sdk.JSONResponse(http.StatusOK, response)
		}
	}

	// No results found
	return sdk.JSONResponse(http.StatusOK, map[string]any{
		"input_type": "IMDB_LOOKUP",
		"query":      imdbID,
		"type":       mediaType,
		"message":    "IMDb ID not found in TMDb",
		"success":    false,
	})
}

// handleTitleSearch performs text search and enriches top results with IMDb IDs.
func (p *TMDbPlugin) handleTitleSearch(ctx context.Context, client *Client, mediaType, query string, response *LookupResponse) (*sdk.HTTPResponse, error) {
	const maxEnrichResults = 5 // Enrich top 5 results with IMDb IDs

	switch mediaType {
	case "movie":
		searchResp, err := client.SearchMovies(ctx, query, 0)
		if err != nil {
			p.recordError()
			return sdk.JSONError(http.StatusInternalServerError, fmt.Sprintf("movie search failed: %v", err))
		}

		// Enrich top results with IMDb IDs
		enriched := make([]EnrichedMovieSearchResult, 0, min(len(searchResp.Results), maxEnrichResults))
		for i, result := range searchResp.Results {
			if i >= maxEnrichResults {
				break
			}
			imdbID := ""
			if extIDs, err := client.GetMovieExternalIDs(ctx, result.ID); err == nil && extIDs != nil {
				imdbID = extIDs.IMDbID
			} else if err != nil {
				p.logger.Debug("failed to fetch external IDs for movie", "tmdb_id", result.ID, "error", err)
			}
			enriched = append(enriched, EnrichedMovieSearchResult{
				MovieSearchResult: result,
				IMDbID:            imdbID,
			})
		}
		response.MovieResults = enriched

	case "tv", "tv_show":
		searchResp, err := client.SearchTV(ctx, query, 0)
		if err != nil {
			p.recordError()
			return sdk.JSONError(http.StatusInternalServerError, fmt.Sprintf("TV search failed: %v", err))
		}

		// Enrich top results with IMDb IDs
		enriched := make([]EnrichedTVSearchResult, 0, min(len(searchResp.Results), maxEnrichResults))
		for i, result := range searchResp.Results {
			if i >= maxEnrichResults {
				break
			}
			imdbID := ""
			if extIDs, err := client.GetTVExternalIDs(ctx, result.ID); err == nil && extIDs != nil {
				imdbID = extIDs.IMDbID
			} else if err != nil {
				p.logger.Debug("failed to fetch external IDs for TV", "tmdb_id", result.ID, "error", err)
			}
			enriched = append(enriched, EnrichedTVSearchResult{
				TVSearchResult: result,
				IMDbID:         imdbID,
			})
		}
		response.TVResults = enriched

	default:
		return sdk.JSONError(http.StatusBadRequest, "invalid type, must be 'movie' or 'tv'")
	}

	response.Query = query
	response.Type = mediaType
	response.InputType = "TITLE_SEARCH"

	return sdk.JSONResponse(http.StatusOK, response)
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// truncate shortens a string to maxLen characters, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
