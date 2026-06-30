package jellyfin

import (
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mantonx/viewra/internal/api/handlers"
	appauth "github.com/mantonx/viewra/internal/application/auth"
	appHome "github.com/mantonx/viewra/internal/application/home"
	appimages "github.com/mantonx/viewra/internal/application/images"
	applibrary "github.com/mantonx/viewra/internal/application/library"
	appmedia "github.com/mantonx/viewra/internal/application/media"
	appmovies "github.com/mantonx/viewra/internal/application/movies"
	appprogress "github.com/mantonx/viewra/internal/application/progress"
	appratings "github.com/mantonx/viewra/internal/application/ratings"
	appSearch "github.com/mantonx/viewra/internal/application/search"
	apptv "github.com/mantonx/viewra/internal/application/tv"
	"github.com/mantonx/viewra/internal/domain/images"
	"github.com/mantonx/viewra/internal/infrastructure/auth"
)

// Handler implements Jellyfin-compatible API endpoints by wrapping ViewRA's services.
type Handler struct {
	getMedia        appmedia.GetMediaExecutor
	listMedia       appmedia.ListMediaExecutor
	libraryService  *applibrary.LibraryService
	moviesList      appmovies.ListMoviesExecutor
	moviesGet       appmovies.GetMovieExecutor
	moviesSearch    appmovies.SearchMoviesExecutor
	tvListShows     apptv.ListTVShowsExecutor
	tvGetShow       apptv.GetTVShowExecutor
	tvListEpisodes  apptv.ListTVEpisodesExecutor
	tvGetEpisode    apptv.GetTVEpisodeExecutor
	tvNextEpisode   apptv.GetNextEpisodeExecutor
	progressSvc     *appprogress.Service
	imagesGet       appimages.GetImageExecutor
	imagesGetMedia  appimages.GetMediaImagesExecutor
	imagesGetEntity appimages.GetEntityImagesExecutor
	searchService   *appSearch.Service
	authService     *appauth.Service
	tokenService    *auth.TokenService
	streamHandler   *handlers.StreamHandler
	transcodeH      *handlers.TranscodeHandler
	imagesH         *handlers.ImagesHandler
	getTracks       appmedia.GetTracksExecutor
	ratingsService  *appratings.Service
	homeService     *appHome.Service
	logger          *slog.Logger
}

// NewHandler creates a new Jellyfin-compatible API handler.
func NewHandler(
	getMedia appmedia.GetMediaExecutor,
	listMedia appmedia.ListMediaExecutor,
	libraryService *applibrary.LibraryService,
	moviesList appmovies.ListMoviesExecutor,
	moviesGet appmovies.GetMovieExecutor,
	moviesSearch appmovies.SearchMoviesExecutor,
	tvListShows apptv.ListTVShowsExecutor,
	tvGetShow apptv.GetTVShowExecutor,
	tvListEpisodes apptv.ListTVEpisodesExecutor,
	tvGetEpisode apptv.GetTVEpisodeExecutor,
	tvNextEpisode apptv.GetNextEpisodeExecutor,
	progressSvc *appprogress.Service,
	imagesGet appimages.GetImageExecutor,
	imagesGetMedia appimages.GetMediaImagesExecutor,
	imagesGetEntity appimages.GetEntityImagesExecutor,
	searchService *appSearch.Service,
	authService *appauth.Service,
	tokenService *auth.TokenService,
	streamHandler *handlers.StreamHandler,
	transcodeH *handlers.TranscodeHandler,
	imagesH *handlers.ImagesHandler,
	getTracks appmedia.GetTracksExecutor,
	ratingsService *appratings.Service,
	homeService *appHome.Service,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		getMedia:        getMedia,
		listMedia:       listMedia,
		libraryService:  libraryService,
		moviesList:      moviesList,
		moviesGet:       moviesGet,
		moviesSearch:    moviesSearch,
		tvListShows:     tvListShows,
		tvGetShow:       tvGetShow,
		tvListEpisodes:  tvListEpisodes,
		tvGetEpisode:    tvGetEpisode,
		tvNextEpisode:   tvNextEpisode,
		progressSvc:     progressSvc,
		imagesGet:       imagesGet,
		imagesGetMedia:  imagesGetMedia,
		imagesGetEntity: imagesGetEntity,
		searchService:   searchService,
		authService:     authService,
		tokenService:    tokenService,
		streamHandler:   streamHandler,
		transcodeH:      transcodeH,
		imagesH:         imagesH,
		getTracks:       getTracks,
		ratingsService:  ratingsService,
		homeService:     homeService,
		logger:          logger,
	}
}

// RegisterRoutes registers all Jellyfin-compatible API routes on the given Gin engine.
// These routes are registered at root level to match the Jellyfin API spec.
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	jf := r.Group("")
	jf.Use(h.jellyfinRequestLogger())
	jf.Use(h.jellyfinTokenMiddleware())

	jf.POST("/Users/AuthenticateByName", h.AuthenticateByName)
	jf.POST("/Users/AuthenticateWithQuickConnect", h.AuthenticateWithQuickConnect)
	jf.GET("/Users/New", h.CreateUser)
	jf.GET("/Users/Public", h.PublicUsers)
	jf.GET("/System/Info/Public", h.SystemInfoPublic)
	jf.GET("/QuickConnect/Enabled", h.QuickConnectEnabled)
	jf.POST("/QuickConnect/Initiate", h.QuickConnectInitiate)
	jf.GET("/QuickConnect/Initiate", h.QuickConnectInitiate)
	jf.GET("/QuickConnect/Connect", h.QuickConnectConnect)
	jf.GET("/Branding/Configuration", h.BrandingConfiguration)
	jf.GET("/System/Ping", h.SystemPing)

	auth := jf.Group("")
	auth.Use(h.requireJellyfinAuth())
	{
		auth.GET("/System/Info", h.SystemInfo)
		auth.GET("/System/Configuration", h.SystemConfiguration)
		auth.GET("/Users/Me", h.GetCurrentUser)
		auth.GET("/Users/:userId", h.GetUser)
		auth.GET("/DisplayPreferences/usersettings", h.DisplayPreferences)
		auth.GET("/DisplayPreferences/:id", h.DisplayPreferences)

		auth.GET("/Users/:userId/Views", h.GetViews)
		auth.GET("/Users/:userId/Items", h.GetItems)
		auth.GET("/Users/:userId/Items/:itemId", h.GetItem)
		auth.GET("/Users/:userId/Items/Resume", h.GetResumeItems)
		auth.GET("/Users/:userId/Items/Latest", h.GetLatestItems)

		auth.GET("/Shows/NextUp", h.GetNextUp)
		auth.GET("/Shows/:seriesId/Seasons", h.GetSeasons)
		auth.GET("/Shows/:seriesId/Episodes", h.GetEpisodes)

		auth.POST("/Items/:itemId/PlaybackInfo", h.GetPlaybackInfo)
		auth.GET("/Items/:itemId/PlaybackInfo", h.GetPlaybackInfo)
		auth.GET("/Items/:itemId/Similar", h.GetSimilarItems)
		auth.GET("/Items/:itemId/Ancestors", h.GetAncestors)
		auth.GET("/Items/:itemId/ThemeMedia", h.GetThemeMedia)

		auth.POST("/Sessions/Playing", h.StartPlayback)
		auth.POST("/Sessions/Playing/Progress", h.ReportPlaybackProgress)
		auth.POST("/Sessions/Playing/Stopped", h.StopPlayback)
		auth.POST("/Sessions/Capabilities/Full", h.ReportCapabilities)

		auth.POST("/Users/:userId/PlayedItems/:itemId", h.MarkPlayed)
		auth.DELETE("/Users/:userId/PlayedItems/:itemId", h.MarkUnplayed)
		auth.POST("/Users/:userId/FavoriteItems/:itemId", h.Favorite)
		auth.DELETE("/Users/:userId/FavoriteItems/:itemId", h.Unfavorite)

	// Image routes - public (no auth required, like Jellyfin)
	jf.GET("/Items/:itemId/Images", h.GetItemImageInfo)
	jf.GET("/Items/:itemId/Images/:imageType", h.GetItemImage)
	jf.GET("/Items/:itemId/Images/:imageType/:index", h.GetItemImageByIndex)

		auth.POST("/Sessions/Logout", h.Logout)

		auth.GET("/Search/Hints", h.GetSearchHints)

		auth.POST("/DisplayPreferences/:id", h.UpdateDisplayPreferences)

		auth.GET("/Collections", h.GetCollections)
		auth.GET("/Collections/:collectionId/Items", h.GetCollectionItems)

		auth.GET("/Users/:userId/Suggestions", h.GetSuggestions)
		auth.GET("/Items/:itemId/SpecialFeatures", h.GetSpecialFeatures)
		auth.GET("/Items/:itemId/AdditionalParts", h.GetAdditionalParts)
		auth.GET("/Items/:itemId/LocalTrailers", h.GetLocalTrailers)
		auth.GET("/Items/:itemId/Intros", h.GetIntros)

		auth.GET("/LiveTv/GuideInfo", h.LiveTvGuideInfo)
		auth.GET("/LiveTv/RecommendedPrograms", h.LiveTvRecommendedPrograms)

		auth.GET("/Notifications/Summary", h.GetNotificationSummary)
		auth.GET("/Notifications/Types", h.GetNotificationTypes)

		auth.GET("/System/Logs", h.GetSystemLogs)
		auth.GET("/Packages", h.GetPackages)

		auth.GET("/Genres", h.GetGenres)
		auth.GET("/Persons", h.GetPersons)

		// Live TV stubs
		auth.GET("/LiveTv/Channels", h.LiveTvChannels)
		auth.GET("/LiveTv/Programs", h.LiveTvPrograms)
		auth.GET("/LiveTv/Recordings", h.LiveTvEmpty)
		auth.GET("/LiveTv/Timers", h.LiveTvEmpty)
		auth.GET("/LiveTv/SeriesTimers", h.LiveTvEmpty)

		// Sessions
		auth.GET("/Sessions", h.GetSessions)

		// Plugin and home screen stubs
		auth.GET("/Plugins", h.GetPlugins)
		auth.GET("/HomeScreen/Meta", h.HomeScreenMeta)
		auth.GET("/HomeScreen/Sections", h.HomeScreenSections)
		auth.GET("/HomeScreen/Section/:sectionType", h.HomeScreenSection)
		auth.GET("/MediaSegments/:itemId", h.GetMediaSegments)

		// Video streaming routes
		auth.GET("/Videos/:id/stream", h.StreamVideo)
		auth.GET("/Videos/:id/stream.:container", h.StreamVideoContainer)
		auth.GET("/Videos/:id/master.m3u8", h.ServeMasterPlaylist)
		auth.GET("/Videos/:id/hls/:quality/playlist.m3u8", h.ServePlaylist)
		auth.GET("/Videos/:id/hls/:quality/:filename", h.ServeHLSSegment)

		// Subtitle streaming
		auth.GET("/Videos/:id/:mediaSourceId/Subtitles/:index/Stream.vtt", h.StreamSubtitleVTT)
		auth.GET("/Videos/:id/:mediaSourceId/Subtitles/:index/Stream.ass", h.StreamSubtitleASS)

		// Audio streaming
		auth.GET("/Audio/:id/stream", h.StreamAudio)
		auth.GET("/Audio/:id/stream.:container", h.StreamAudioContainer)
		auth.GET("/Audio/:id/master.m3u8", h.AudioMasterPlaylist)
		auth.GET("/Audio/:id/hls/:quality/:filename", h.AudioHLSSegment)
	}

}

// jellyfinRequestLogger logs every incoming Jellyfin API request for debugging.
func (h *Handler) jellyfinRequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		h.logger.Debug("jellyfin: request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"user-agent", c.Request.UserAgent(),
		)
		c.Next()
	}
}

// jellyfinTokenMiddleware extracts a Jellyfin/Emby auth token from the request
// and stores it in the context for downstream handlers.
func (h *Handler) jellyfinTokenMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractJellyfinToken(c)
		if token != "" {
			c.Set("jellyfin_token", token)
		}
		c.Next()
	}
}

// requireJellyfinAuth returns middleware that validates the Jellyfin access token.
func (h *Handler) requireJellyfinAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractJellyfinToken(c)
		if token == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "Missing authorization token"})
			return
		}

		claims, err := h.tokenService.ValidateAccessToken(token)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "Invalid or expired token"})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("is_admin", claims.IsAdmin)
		c.Set("claims", claims)
		c.Next()
	}
}

// extractJellyfinToken extracts the auth token from Jellyfin-style auth headers.
// Handles formats:
//   Authorization: MediaBrowser Token="<token>"
//   Authorization: MediaBrowser Client="...", Device="...", Token="<token>"
//   X-Emby-Token: <token>
//   ?api_key=<token>
func extractJellyfinToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if strings.HasPrefix(authHeader, "MediaBrowser ") {
		tokenStart := strings.Index(authHeader, "Token=\"")
		if tokenStart >= 0 {
			tokenStart += len("Token=\"")
			tokenEnd := strings.Index(authHeader[tokenStart:], "\"")
			if tokenEnd > 0 {
				return authHeader[tokenStart : tokenStart+tokenEnd]
			}
		}
	}
	embyToken := c.GetHeader("X-Emby-Token")
	if embyToken != "" {
		return embyToken
	}
	apiKey := c.Query("api_key")
	if apiKey != "" {
		return apiKey
	}
	apiKey = c.Query("ApiKey")
	if apiKey != "" {
		return apiKey
	}
	return ""
}

// --- Jellyfin DTOs ---

type SystemInfo struct {
	Id                   string `json:"Id"`
	SystemUpdateLevel    string `json:"SystemUpdateLevel"`
	OperatingSystem      string `json:"OperatingSystem"`
	ServerName           string `json:"ServerName"`
	Version              string `json:"Version"`
	ProductName          string `json:"ProductName"`
	StartupWizardCompleted bool `json:"StartupWizardCompleted"`
	WebPath              string `json:"WebPath,omitempty"`
}

type AuthenticationResult struct {
	User        *UserDto `json:"User"`
	SessionInfo any      `json:"SessionInfo"`
	AccessToken string   `json:"AccessToken"`
	ServerId    string   `json:"ServerId"`
}

type UserDto struct {
	Name                      string `json:"Name"`
	ServerId                  string `json:"ServerId"`
	Id                        string `json:"Id"`
	PrimaryImageTag           string `json:"PrimaryImageTag,omitempty"`
	HasPassword               bool   `json:"HasPassword"`
	HasConfiguredPassword     bool   `json:"HasConfiguredPassword"`
	EnableAutoLogin           bool   `json:"EnableAutoLogin"`
	LastLoginDate             string `json:"LastLoginDate,omitempty"`
	LastActivityDate          string `json:"LastActivityDate,omitempty"`
	Configuration             any    `json:"Configuration"`
	Policy                    any    `json:"Policy"`
}

type BaseItemDto struct {
	Name                string            `json:"Name"`
	Id                  string            `json:"Id"`
	ServerId            string            `json:"ServerId"`
	Type                string            `json:"Type"`
	CollectionType      string            `json:"CollectionType,omitempty"`
	MediaType           string            `json:"MediaType"`
	ParentId            string            `json:"ParentId,omitempty"`
	Path                string            `json:"Path,omitempty"`
	SortName            string            `json:"SortName,omitempty"`
	Overview            string            `json:"Overview,omitempty"`
	Taglines            []string          `json:"Taglines,omitempty"`
	Genres              []string          `json:"Genres,omitempty"`
	CommunityRating     *float32          `json:"CommunityRating,omitempty"`
	RunTimeTicks        *int64            `json:"RunTimeTicks,omitempty"`
	ProductionYear      *int              `json:"ProductionYear,omitempty"`
	IndexNumber         *int              `json:"IndexNumber,omitempty"`
	ParentIndexNumber   *int              `json:"ParentIndexNumber,omitempty"`
	SeriesName          string            `json:"SeriesName,omitempty"`
	SeriesId            string            `json:"SeriesId,omitempty"`
	SeasonId            string            `json:"SeasonId,omitempty"`
	PremiereDate        string            `json:"PremiereDate,omitempty"`
	DateCreated         string            `json:"DateCreated,omitempty"`
	ProviderIds         map[string]string `json:"ProviderIds,omitempty"`
	IsFolder            bool              `json:"IsFolder"`
	ChildCount          *int              `json:"ChildCount,omitempty"`
	ImageTags           map[string]string `json:"ImageTags,omitempty"`
	BackdropImageTags   []string          `json:"BackdropImageTags,omitempty"`
	BackdropCount       int               `json:"BackdropCount,omitempty"`
	Width               int               `json:"Width,omitempty"`
	Height              int               `json:"Height,omitempty"`
	VideoType           string            `json:"VideoType,omitempty"`
	Container           string            `json:"Container,omitempty"`
	MediaSources        []MediaSource     `json:"MediaSources,omitempty"`
	MediaStreams        []MediaStream     `json:"MediaStreams,omitempty"`
	UserData            *UserDataDto      `json:"UserData,omitempty"`
	SeriesThumbImageTag string            `json:"SeriesThumbImageTag,omitempty"`
	SeriesPrimaryImageTag string          `json:"SeriesPrimaryImageTag,omitempty"`
	SeasonName          string            `json:"SeasonName,omitempty"`
	LocationType        string            `json:"LocationType"`
}

type UserDataDto struct {
	Played        bool    `json:"Played"`
	PlayedPercentage *float64 `json:"PlayedPercentage,omitempty"`
	PlaybackPositionTicks *int64 `json:"PlaybackPositionTicks,omitempty"`
	IsFavorite    bool    `json:"IsFavorite"`
	UnplayedItemCount *int `json:"UnplayedItemCount,omitempty"`
	PlayCount     int     `json:"PlayCount"`
	Key           string  `json:"Key"`
}

type MediaSource struct {
	Id                     string        `json:"Id"`
	Name                   string        `json:"Name"`
	ETag                   string        `json:"ETag,omitempty"`
	Type                   string        `json:"Type"`
	MediaStreams           []MediaStream `json:"MediaStreams"`
	Container              string        `json:"Container,omitempty"`
	Path                   string        `json:"Path,omitempty"`
	RunTimeTicks           int64         `json:"RunTimeTicks"`
	Size                   int64         `json:"Size,omitempty"`
	DirectStreamUrl        string        `json:"DirectStreamUrl,omitempty"`
	TranscodingUrl         string        `json:"TranscodingUrl,omitempty"`
	TranscodingSubProtocol string        `json:"TranscodingSubProtocol,omitempty"`
	SupportsDirectPlay     bool          `json:"SupportsDirectPlay"`
	SupportsDirectStream   bool          `json:"SupportsDirectStream"`
	SupportsTranscoding    bool          `json:"SupportsTranscoding"`
	VideoType              string        `json:"VideoType,omitempty"`
	MediaAttachments       []any         `json:"MediaAttachments,omitempty"`
	Formats                []string      `json:"Formats,omitempty"`
	Bitrate                int64         `json:"Bitrate,omitempty"`
	DefaultAudioStreamIndex *int         `json:"DefaultAudioStreamIndex,omitempty"`
	DefaultSubtitleStreamIndex *int      `json:"DefaultSubtitleStreamIndex,omitempty"`
	Genres                 []string      `json:"Genres,omitempty"`
}

type MediaStream struct {
	Codec                  string `json:"Codec,omitempty"`
	CodecTag               string `json:"CodecTag,omitempty"`
	Language               string `json:"Language,omitempty"`
	TimeBase               string `json:"TimeBase,omitempty"`
	CodecTimeBase          string `json:"CodecTimeBase,omitempty"`
	Title                  string `json:"Title,omitempty"`
	DisplayTitle           string `json:"DisplayTitle,omitempty"`
	IsDefault              bool   `json:"IsDefault"`
	IsForced               bool   `json:"IsForced"`
	Index                  int    `json:"Index"`
	Type                   string `json:"Type"`
	AvgFrameRate           float64 `json:"AvgFrameRate,omitempty"`
	RealFrameRate          float64 `json:"RealFrameRate,omitempty"`
	Profile                string `json:"Profile,omitempty"`
	AspectRatio            string `json:"AspectRatio,omitempty"`
	Width                  int    `json:"Width,omitempty"`
	Height                 int    `json:"Height,omitempty"`
	IsInterlaced           bool   `json:"IsInterlaced"`
	BitRate                int64  `json:"BitRate,omitempty"`
	ChannelLayout          string `json:"ChannelLayout,omitempty"`
	Channels               int    `json:"Channels,omitempty"`
	SampleRate             int    `json:"SampleRate,omitempty"`
	IsAVC                  bool   `json:"IsAVC,omitempty"`
}

type PlaybackInfoResponse struct {
	MediaSources []MediaSource `json:"MediaSources"`
	PlaySessionId string       `json:"PlaySessionId"`
	ErrorCode     string       `json:"ErrorCode,omitempty"`
}

type PlaybackStartInfo struct {
	CanSeek        bool   `json:"CanSeek"`
	ItemId         string `json:"ItemId"`
	MediaSourceId  string `json:"MediaSourceId"`
	PlayMethod     string `json:"PlayMethod"`
	PlaySessionId  string `json:"PlaySessionId"`
	PositionTicks  int64  `json:"PositionTicks,omitempty"`
}

type PlaybackProgressInfo struct {
	CanSeek        bool   `json:"CanSeek"`
	ItemId         string `json:"ItemId"`
	MediaSourceId  string `json:"MediaSourceId"`
	PlayMethod     string `json:"PlayMethod"`
	PlaySessionId  string `json:"PlaySessionId"`
	PositionTicks  int64  `json:"PositionTicks,omitempty"`
	IsPaused       bool   `json:"IsPaused"`
}

type PlaybackStopInfo struct {
	ItemId         string `json:"ItemId"`
	MediaSourceId  string `json:"MediaSourceId"`
	PlaySessionId  string `json:"PlaySessionId"`
	PositionTicks  int64  `json:"PositionTicks,omitempty"`
}

type ItemsResult struct {
	Items            []BaseItemDto `json:"Items"`
	TotalRecordCount int           `json:"TotalRecordCount"`
	StartIndex       int           `json:"StartIndex"`
}

type ImageInfo struct {
	ImageType string `json:"ImageType"`
	ImageIndex *int  `json:"ImageIndex,omitempty"`
	Path      string `json:"Path,omitempty"`
	Width     int    `json:"Width,omitempty"`
	Height    int    `json:"Height,omitempty"`
}

type SearchHintResult struct {
	SearchHints      []SearchHint `json:"SearchHints"`
	TotalRecordCount int          `json:"TotalRecordCount"`
}

type SearchHint struct {
	ItemId          string            `json:"ItemId"`
	Id              string            `json:"Id"`
	Name            string            `json:"Name"`
	Type            string            `json:"Type"`
	MediaType       string            `json:"MediaType"`
	PrimaryImageTag string            `json:"PrimaryImageTag,omitempty"`
	ProductionYear  *int              `json:"ProductionYear,omitempty"`
	RunTimeTicks    *int64            `json:"RunTimeTicks,omitempty"`
	ProviderIds     map[string]string `json:"ProviderIds,omitempty"`
}

type NotificationSummary struct {
	Unread                 int    `json:"Unread"`
	MaxUnreadNotificationLevel string `json:"MaxUnreadNotificationLevel"`
}

// --- Helpers ---

func secondsToTicks(seconds float64) int64 {
	return int64(seconds * 10000000)
}

func ticksToSeconds(ticks int64) float64 {
	return float64(ticks) / 10000000
}

func fmtTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func idStr(id int64) string {
	return strconv.FormatInt(id, 10)
}

func parseInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// itemImageTags returns the image tags map for an item.
func (h *Handler) itemImageTags(c *gin.Context, mediaID int64, mediaType string) map[string]string {
	tags := make(map[string]string)
	imgs, err := h.imagesGetMedia.Execute(c.Request.Context(), int(mediaID))
	if err != nil || imgs == nil || len(imgs.Images) == 0 {
		return tags
	}
	for _, img := range imgs.Images {
		jfType := mapViewraImageTypeToJellyfin(img.ImageType)
		if jfType != "" {
			if img.FileHash != nil {
				tags[jfType] = *img.FileHash
			}
			if jfType == "Backdrop" {
				continue
			}
			if _, ok := tags["Primary"]; !ok {
				tags["Primary"] = tags[jfType]
			}
		}
	}
	return tags
}

// backdropCount returns the number of backdrop/fanart images.
func (h *Handler) backdropCount(c *gin.Context, mediaID int64) int {
	imgs, err := h.imagesGetMedia.Execute(c.Request.Context(), int(mediaID))
	if err != nil || imgs == nil {
		return 0
	}
	count := 0
	for _, img := range imgs.Images {
		if img.ImageType == string(images.ImageTypeFanart) || img.ImageType == string(images.ImageTypeBackdrop) {
			count++
		}
	}
	return count
}

// getUserData returns user data (watched status, progress, favorite) for an item.
func (h *Handler) getUserData(c *gin.Context, itemID int64, itemType string) *UserDataDto {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		userID = 1
	}

	progress, err := h.progressSvc.GetProgress(c.Request.Context(), userID, itemID)
	if err != nil {
		return &UserDataDto{
			PlayCount: 0,
			Key:       idStr(itemID),
		}
	}

	playedPct := progress.ProgressPercentage
	positionTicks := secondsToTicks(progress.ProgressSeconds)

	isFavorite := false
	if h.ratingsService != nil {
		entityType := mapViewraTypeForRatings(itemType)
		rating, err := h.ratingsService.Get(c.Request.Context(), idStr(userID), entityType, itemID)
		if err == nil && rating != nil && rating.Rating == "favorite" {
			isFavorite = true
		}
	}

	return &UserDataDto{
		Played:                progress.IsWatched,
		PlayedPercentage:      &playedPct,
		PlaybackPositionTicks: &positionTicks,
		IsFavorite:            isFavorite,
		PlayCount:             1,
		Key:                   idStr(itemID),
	}
}

func mapViewraTypeForRatings(itemType string) string {
	switch itemType {
	case "Movie":
		return "movie"
	case "Series", "Season":
		return "tv_show"
	case "Episode":
		return "tv_episode"
	default:
		return "movie"
	}
}
