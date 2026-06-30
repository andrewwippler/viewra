package jellyfin

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	appmedia "github.com/mantonx/viewra/internal/application/media"
	appmovies "github.com/mantonx/viewra/internal/application/movies"
	"github.com/mantonx/viewra/internal/domain/common"
	"github.com/mantonx/viewra/internal/domain/images"
	"github.com/mantonx/viewra/internal/domain/search"
)

// GetItems handles GET /Users/:userId/Items
func (h *Handler) GetItems(c *gin.Context) {
	parentIDStr := c.Query("ParentId")
	includeTypes := strings.Split(c.DefaultQuery("IncludeItemTypes", ""), ",")
	searchTerm := c.Query("SearchTerm")
	sortBy := c.DefaultQuery("SortBy", "SortName")
	sortOrder := c.DefaultQuery("SortOrder", "Ascending")
	startIndex, _ := strconv.Atoi(c.DefaultQuery("StartIndex", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("Limit", "100"))
	seasonStr := c.Query("Season")
	recursive := c.DefaultQuery("Recursive", "false") == "true"
	filters := c.Query("Filters")
	isPlayed := c.Query("IsPlayed")
	isFavorite := c.Query("IsFavorite")
	minCommunityRating, _ := strconv.ParseFloat(c.DefaultQuery("MinCommunityRating", "0"), 64)

	var parentID int64
	if parentIDStr != "" {
		parentID, _ = parseInt64(parentIDStr)
	}

	pagination := &common.PaginationParams{
		Limit:  limit,
		Offset: startIndex,
	}

	if searchTerm != "" {
		h.searchItems(c, searchTerm, parentID, includeTypes, pagination)
		return
	}

	hasMovie := contains(includeTypes, "Movie")
	hasEpisode := contains(includeTypes, "Episode")
	hasSeries := contains(includeTypes, "Series")
	hasSeason := contains(includeTypes, "Season")

	if seasonStr != "" && parentID > 0 {
		seasonNum, err := strconv.Atoi(seasonStr)
		if err == nil {
			h.listEpisodesBySeason(c, parentID, seasonNum, pagination)
			return
		}
	}

	if hasEpisode && parentID > 0 {
		h.listEpisodesForShow(c, parentID, pagination)
		return
	}

	if hasSeries && parentID > 0 {
		h.listTVShows(c, parentID, pagination)
		return
	}

	if hasSeason && parentID > 0 {
		h.listSeasonsForShow(c, parentID)
		return
	}

	if hasMovie && parentID > 0 {
		h.listMovies(c, parentID, sortBy, sortOrder, pagination, filters, isPlayed, isFavorite, minCommunityRating)
		return
	}

	if parentID > 0 {
		// When no IncludeItemTypes specified, detect library type and route accordingly
		if len(includeTypes) == 0 || includeTypes[0] == "" {
			libType := h.getLibraryType(parentID)
			switch libType {
			case "tv":
				h.listTVShows(c, parentID, pagination)
				return
			case "music":
				h.listAllItems(c, []string{"Audio"}, pagination, "", "", "", 0)
				return
			default:
				h.listMovies(c, parentID, sortBy, sortOrder, pagination, filters, isPlayed, isFavorite, minCommunityRating)
				return
			}
		}
		h.listLibraryItems(c, parentID, includeTypes, sortBy, sortOrder, pagination)
		return
	}

	if recursive && parentID == 0 {
		h.listAllItems(c, includeTypes, pagination, filters, isPlayed, isFavorite, minCommunityRating)
		return
	}

	c.JSON(http.StatusOK, ItemsResult{
		Items:            []BaseItemDto{},
		TotalRecordCount: 0,
		StartIndex:       startIndex,
	})
}

// GetItem handles GET /Users/:userId/Items/:itemId
func (h *Handler) GetItem(c *gin.Context) {
	itemID, err := parseInt64(c.Param("itemId"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	mediaResp, err := h.getMedia.Execute(c.Request.Context(), itemID)
	if err == nil {
		c.JSON(http.StatusOK, h.mediaToBaseItem(c, &mediaResp))
		return
	}

	movieResp, err2 := h.moviesGet.Execute(c.Request.Context(), itemID)
	if err2 == nil {
		c.JSON(http.StatusOK, h.movieToBaseItem(c, movieResp))
		return
	}

	showResp, err4 := h.tvGetShow.Execute(c.Request.Context(), itemID)
	if err4 == nil {
		c.JSON(http.StatusOK, h.showToBaseItem(c, showResp))
		return
	}

	epResp, err3 := h.tvGetEpisode.Execute(c.Request.Context(), itemID)
	if err3 == nil {
		c.JSON(http.StatusOK, h.episodeToBaseItem(c, epResp, 0))
		return
	}

	c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Item not found"})
}

// GetResumeItems handles GET /Users/:userId/Items/Resume
func (h *Handler) GetResumeItems(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		userID = 1
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("Limit", "40"))

	progress, err := h.progressSvc.ListInProgress(c.Request.Context(), userID, limit, 0)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to get resume items"})
		return
	}

	items := make([]BaseItemDto, 0, len(progress.Progress))
	for _, p := range progress.Progress {
		mediaResp, err := h.getMedia.Execute(c.Request.Context(), p.MediaID)
		if err != nil {
			continue
		}
		item := h.mediaToBaseItem(c, &mediaResp)
		playedPct := p.ProgressPercentage
		positionTicks := secondsToTicks(p.ProgressSeconds)
		item.UserData = &UserDataDto{
			Played:                p.IsWatched,
			PlayedPercentage:      &playedPct,
			PlaybackPositionTicks: &positionTicks,
			PlayCount:             1,
		}
		items = append(items, item)
	}

	c.JSON(http.StatusOK, ItemsResult{
		Items:            items,
		TotalRecordCount: len(items),
		StartIndex:       0,
	})
}

// GetLatestItems handles GET /Users/:userId/Items/Latest
func (h *Handler) GetLatestItems(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("Limit", "20"))

	mediaResp, err := h.listMedia.ExecuteAll(c.Request.Context())
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to list media"})
		return
	}

	items := make([]BaseItemDto, 0, limit)
	for i := len(mediaResp.Media) - 1; i >= 0 && len(items) < limit; i-- {
		m := mediaResp.Media[i]
		items = append(items, h.mediaToBaseItem(c, &m))
	}

	c.JSON(http.StatusOK, items)
}

func (h *Handler) listAllItems(c *gin.Context, includeTypes []string, pagination *common.PaginationParams, filters, isPlayed, isFavorite string, minCommunityRating float64) {
	hasMovie := len(includeTypes) == 0 || includeTypes[0] == "" || contains(includeTypes, "Movie")
	hasSeries := contains(includeTypes, "Series")
	hasEpisode := contains(includeTypes, "Episode")
	isUnplayed := filters == "IsUnplayed" || isPlayed == "false"
	onlyFavorite := isFavorite == "true"

	userID := c.GetInt64("user_id")
	if userID == 0 {
		userID = 1
	}
	_ = userID

	items := make([]BaseItemDto, 0)

	if hasMovie {
		libs, err := h.libraryService.List(c.Request.Context())
		if err == nil {
			for _, lib := range libs.Libraries {
				moviesResp, err := h.moviesList.Execute(c.Request.Context(), lib.ID)
				if err != nil {
					continue
				}
				for _, m := range moviesResp.Movies {
					itemUserData := h.getUserData(c, m.ID, "Movie")
					if onlyFavorite && (itemUserData == nil || !itemUserData.IsFavorite) {
						continue
					}
					if isUnplayed && itemUserData != nil && itemUserData.Played {
						continue
					}
					if minCommunityRating > 0 && m.Rating < float32(minCommunityRating) {
						continue
					}
					item := h.movieToBaseItem(c, &m)
					items = append(items, item)
				}
			}
		}
	}

	if hasSeries || hasEpisode {
		libs, err := h.libraryService.List(c.Request.Context())
		if err == nil {
			for _, lib := range libs.Libraries {
				if lib.Type != "tv" {
					continue
				}
				if hasSeries {
					showsResp, err := h.tvListShows.Execute(c.Request.Context(), lib.ID)
					if err == nil {
						for _, s := range showsResp.Shows {
							showDetail, detailErr := h.tvGetShow.Execute(c.Request.Context(), s.ID)
							if detailErr == nil {
								items = append(items, h.showToBaseItem(c, showDetail))
							}
						}
					}
				}
				if hasEpisode {
					epsResp, err := h.tvListEpisodes.ExecuteByLibrary(c.Request.Context(), lib.ID)
					if err == nil {
						for _, ep := range epsResp.Episodes {
							item := h.episodeToBaseItem(c, &ep, 0)
							items = append(items, item)
						}
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, ItemsResult{
		Items:            items,
		TotalRecordCount: len(items),
		StartIndex:       pagination.Offset,
	})
}

func (h *Handler) searchItems(c *gin.Context, query string, parentID int64, types []string, pagination *common.PaginationParams) {
	_ = parentID

	searchResp, err := h.searchService.Search(c.Request.Context(), &search.Request{
		Query: query,
		Limit: pagination.Limit,
	})
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}

	items := make([]BaseItemDto, 0, len(searchResp.Results))
	for _, r := range searchResp.Results {
		jellyfinType := mapSearchTypeToJellyfin(r.MediaType)
		if len(types) > 0 && types[0] != "" && !contains(types, jellyfinType) {
			continue
		}
		items = append(items, BaseItemDto{
			Name:          r.Title,
			Id:            idStr(r.ID),
			ServerId:      "viewra",
			Type:          jellyfinType,
			ProductionYear: &r.Year,
			LocationType:  "FileSystem",
		})
	}

	c.JSON(http.StatusOK, ItemsResult{
		Items:            items,
		TotalRecordCount: len(items),
		StartIndex:       0,
	})
}

func (h *Handler) listMovies(c *gin.Context, libraryID int64, sortBy, sortOrder string, pagination *common.PaginationParams, filters, isPlayed, isFavorite string, minCommunityRating float64) {
	moviesResp, err := h.moviesList.Execute(c.Request.Context(), libraryID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to list movies"})
		return
	}

	items := make([]BaseItemDto, 0, len(moviesResp.Movies))
	for _, m := range moviesResp.Movies {
		if minCommunityRating > 0 && m.Rating < float32(minCommunityRating) {
			continue
		}
		item := h.movieToBaseItem(c, &m)
		items = append(items, item)
	}

	_ = sortBy
	_ = sortOrder

	c.JSON(http.StatusOK, ItemsResult{
		Items:            items,
		TotalRecordCount: len(items),
		StartIndex:       pagination.Offset,
	})
}

// --- Conversion helpers ---

func (h *Handler) mediaToBaseItem(c *gin.Context, m *appmedia.MediaResponse) BaseItemDto {
	ticks := int64(m.Duration) * 10000000
	item := BaseItemDto{
		Name:         m.Title,
		Id:           idStr(m.ID),
		ServerId:     "viewra",
		Type:         mediaTypeToJellyfin(m.Type),
		IsFolder:     false,
		RunTimeTicks: &ticks,
		Width:        m.Width,
		Height:       m.Height,
		Container:    m.ContainerFormat,
		DateCreated:  fmtTime(m.CreatedAt),
		Path:         m.FilePath,
		LocationType: "FileSystem",
	}
	if m.Type == "music_track" {
		item.MediaType = "Audio"
	} else {
		item.MediaType = "Video"
	}
	item.ImageTags = h.itemImageTags(c, m.ID, m.Type)
	item.BackdropCount = h.backdropCount(c, m.ID)
	item.UserData = h.getUserData(c, m.ID, item.Type)
	return item
}

func (h *Handler) movieToBaseItem(c *gin.Context, m *appmovies.MovieResponse) BaseItemDto {
	ticks := int64(m.Duration) * 10000000
	item := BaseItemDto{
		Name:            m.Title,
		Id:              idStr(m.ID),
		ServerId:        "viewra",
		Type:            "Movie",
		IsFolder:         false,
		Overview:        m.Plot,
		Taglines:        ifString(m.Tagline),
		Genres:          m.Genre,
		RunTimeTicks:    &ticks,
		Width:           m.Width,
		Height:          m.Height,
		Container:       m.ContainerFormat,
		ProductionYear:  &m.Year,
		PremiereDate:    fmtTime(m.ReleaseDate),
		DateCreated:     fmtTime(m.CreatedAt),
		Path:            m.FilePath,
		LocationType:    "FileSystem",
	}

	if m.IMDbID != "" {
		item.ProviderIds = map[string]string{"Imdb": m.IMDbID}
	}
	if m.TMDbID > 0 {
		if item.ProviderIds == nil {
			item.ProviderIds = make(map[string]string)
		}
		item.ProviderIds["Tmdb"] = strconv.Itoa(m.TMDbID)
	}

	if m.Rating > 0 {
		f := float32(m.Rating)
		item.CommunityRating = &f
	}
	if m.SortTitle != "" {
		item.SortName = m.SortTitle
	}

	item.ImageTags = h.itemImageTags(c, m.ID, "movie")
	item.BackdropCount = h.backdropCount(c, m.ID)
	item.MediaType = "Video"
	item.UserData = h.getUserData(c, m.ID, "Movie")

	// Build MediaSources from language variants
	if len(m.Variants) > 0 {
		item.MediaSources = make([]MediaSource, 0, len(m.Variants)+1)
		// Primary source
		primaryID := idStr(m.ID)
		container := m.ContainerFormat
		if container == "" {
			container = "mp4"
		}
		item.MediaSources = append(item.MediaSources, MediaSource{
			Id:                     primaryID,
			Name:                   "Original",
			Type:                   "Default",
			Container:              container,
			Path:                   m.FilePath,
			RunTimeTicks:           ticks,
			Size:                   m.FileSize,
			SupportsDirectPlay:     true,
			SupportsDirectStream:   true,
			SupportsTranscoding:    true,
			DirectStreamUrl:        "/Videos/" + primaryID + "/stream." + container + "?Static=true&mediaSourceId=" + primaryID,
			TranscodingUrl:         "/Videos/" + primaryID + "/master.m3u8",
			Bitrate:                m.Bitrate,
		})
		for _, v := range m.Variants {
			vid := idStr(v.ID)
			vContainer := v.ContainerFormat
			if vContainer == "" {
				vContainer = "mp4"
			}
			name := v.Language
			if name == "" {
				name = "Unknown"
			}
			item.MediaSources = append(item.MediaSources, MediaSource{
				Id:                     vid,
				Name:                   name,
				Type:                   "Default",
				Container:              vContainer,
				Path:                   v.FilePath,
				RunTimeTicks:           int64(v.Duration) * 10000000,
				Size:                   v.FileSize,
				SupportsDirectPlay:     true,
				SupportsDirectStream:   true,
				SupportsTranscoding:    true,
				DirectStreamUrl:        "/Videos/" + vid + "/stream." + vContainer + "?Static=true&mediaSourceId=" + vid,
				TranscodingUrl:         "/Videos/" + vid + "/master.m3u8",
				Bitrate:                m.Bitrate,
			})
		}
	}

	return item
}

// --- Pure helpers ---

func mediaTypeToJellyfin(viewraType string) string {
	switch viewraType {
	case "movie":
		return "Movie"
	case "tv_episode":
		return "Episode"
	case "music_track":
		return "Audio"
	default:
		return "Movie"
	}
}

func mapSearchTypeToJellyfin(mediaType string) string {
	switch mediaType {
	case "movie":
		return "Movie"
	case "tv_episode":
		return "Episode"
	case "tv_show":
		return "Series"
	default:
		return "Movie"
	}
}

func mapViewraImageTypeToJellyfin(viewraType string) string {
	switch viewraType {
	case string(images.ImageTypePoster):
		return "Primary"
	case string(images.ImageTypeFanart), string(images.ImageTypeBackdrop):
		return "Backdrop"
	case string(images.ImageTypeBanner):
		return "Banner"
	case string(images.ImageTypeThumb):
		return "Thumb"
	case string(images.ImageTypeClearLogo):
		return "Logo"
	case string(images.ImageTypeLandscape):
		return "Thumb"
	default:
		return ""
	}
}

func mapViewraLibTypeToJellyfin(viewraType string) string {
	switch viewraType {
	case "movies":
		return "movies"
	case "tv":
		return "tvshows"
	case "music":
		return "music"
	default:
		return ""
	}
}

func mapJellyfinImageTypeToViewra(jfType string) string {
	switch strings.ToLower(jfType) {
	case "primary":
		return string(images.ImageTypePoster)
	case "backdrop", "fanart":
		return string(images.ImageTypeFanart)
	case "banner":
		return string(images.ImageTypeBanner)
	case "thumb", "thumbnail":
		return string(images.ImageTypeThumb)
	case "logo":
		return string(images.ImageTypeClearLogo)
	default:
		return ""
	}
}

func contains(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func ifString(s string) []string {
	if s != "" {
		return []string{s}
	}
	return nil
}

func intPtr(i int) *int {
	return &i
}

func parseYearFromDate(dateStr string) *int {
	if len(dateStr) >= 4 {
		year, err := strconv.Atoi(dateStr[:4])
		if err == nil && year > 0 {
			return &year
		}
	}
	return nil
}

// getLibraryType returns the type ("movies", "tv", "music") for a given library ID.
func (h *Handler) getLibraryType(libraryID int64) string {
	libs, err := h.libraryService.List(context.Background())
	if err != nil {
		return "movies" // default fallback
	}
	for _, lib := range libs.Libraries {
		if lib.ID == libraryID {
			return lib.Type
		}
	}
	return "movies"
}
