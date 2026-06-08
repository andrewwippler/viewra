package jellyfin

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	apptv "github.com/mantonx/viewra/internal/application/tv"
	"github.com/mantonx/viewra/internal/domain/common"
)

// GetNextUp handles GET /Shows/NextUp
func (h *Handler) GetNextUp(c *gin.Context) {
	libs, err := h.libraryService.List(c.Request.Context())
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to list libraries"})
		return
	}

	items := make([]BaseItemDto, 0)

	for _, lib := range libs.Libraries {
		if lib.Type != "tv" {
			continue
		}

		showsResp, err := h.tvListShows.Execute(c.Request.Context(), lib.ID)
		if err != nil {
			continue
		}

		for _, show := range showsResp.Shows {
			nextEp, err := h.tvNextEpisode.Execute(c.Request.Context(), show.ID)
			if err != nil || nextEp == nil {
				continue
			}

			item := h.episodeToBaseItem(c, nextEp, show.ID)
			items = append(items, item)
		}
	}

	c.JSON(http.StatusOK, ItemsResult{
		Items:            items,
		TotalRecordCount: len(items),
		StartIndex:       0,
	})
}

// GetSeasons handles GET /Shows/:seriesId/Seasons
func (h *Handler) GetSeasons(c *gin.Context) {
	showID, err := parseInt64(c.Param("seriesId"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid series ID"})
		return
	}

	h.listSeasonsForShow(c, showID)
}

// GetEpisodes handles GET /Shows/:seriesId/Episodes
func (h *Handler) GetEpisodes(c *gin.Context) {
	showID, err := parseInt64(c.Param("seriesId"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid series ID"})
		return
	}

	seasonIDStr := c.Query("seasonId")
	startIndex, _ := strconv.Atoi(c.DefaultQuery("StartIndex", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("Limit", "100"))

	pagination := &common.PaginationParams{
		Limit:  limit,
		Offset: startIndex,
	}

	if seasonIDStr != "" {
		seasonID, err := parseInt64(seasonIDStr)
		if err == nil && seasonID > 0 {
			seasonNum := int(seasonID >> 32)
			if seasonNum > 0 {
				h.listEpisodesBySeason(c, showID, seasonNum, pagination)
				return
			}
		}
	}

	h.listEpisodesForShow(c, showID, pagination)
}

func (h *Handler) listTVShows(c *gin.Context, libraryID int64, pagination *common.PaginationParams) {
	showsResp, err := h.tvListShows.Execute(c.Request.Context(), libraryID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to list TV shows"})
		return
	}

	items := make([]BaseItemDto, 0, len(showsResp.Shows))
	for _, s := range showsResp.Shows {
		ticks := int64(0)
		item := BaseItemDto{
			Name:           s.Title,
			Id:             idStr(s.ID),
			ServerId:       "viewra",
			Type:           "Series",
			MediaType:      "Video",
			IsFolder:        true,
			RunTimeTicks:   &ticks,
			Overview:       s.Plot,
			Genres:         s.Genre,
			ProductionYear: &s.Year,
			LocationType:   "FileSystem",
		}
		if s.IMDbID != "" {
			item.ProviderIds = map[string]string{"Imdb": s.IMDbID}
		}
		if s.TMDbID > 0 {
			if item.ProviderIds == nil {
				item.ProviderIds = make(map[string]string)
			}
			item.ProviderIds["Tmdb"] = strconv.Itoa(s.TMDbID)
		}
		if s.TVDbID > 0 {
			if item.ProviderIds == nil {
				item.ProviderIds = make(map[string]string)
			}
			item.ProviderIds["Tvdb"] = strconv.Itoa(s.TVDbID)
		}
		if s.Rating > 0 {
			f := s.Rating
			item.CommunityRating = &f
		}
		if s.FirstAirDate != "" {
			item.PremiereDate = s.FirstAirDate
		}
		if s.Tagline != "" {
			item.Taglines = ifString(s.Tagline)
		}
		if s.SortTitle != "" {
			item.SortName = s.SortTitle
		}
		item.ImageTags = h.itemImageTags(c, s.ID, "series")
		item.BackdropCount = h.backdropCount(c, s.ID)
		if s.SeasonCount > 0 {
			item.ChildCount = &s.SeasonCount
		}
		item.UserData = h.getUserData(c, s.ID, "Series")
		items = append(items, item)
	}

	c.JSON(http.StatusOK, ItemsResult{
		Items:            items,
		TotalRecordCount: len(items),
		StartIndex:       pagination.Offset,
	})
}

func (h *Handler) listSeasonsForShow(c *gin.Context, showID int64) {
	episodesResp, err := h.tvListEpisodes.ExecuteByShowID(c.Request.Context(), showID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to list episodes"})
		return
	}

	seasonMap := make(map[int]*BaseItemDto)
	for _, ep := range episodesResp.Episodes {
		if _, ok := seasonMap[ep.Season]; !ok {
			seasonID := int64(ep.Season)<<32 | 1
			childCount := 0
			item := BaseItemDto{
				Name:         seasonName("", ep.Season),
				Id:           idStr(seasonID),
				ServerId:     "viewra",
				Type:         "Season",
				IsFolder:      true,
				SeriesId:     idStr(showID),
				IndexNumber:  &ep.Season,
				ChildCount:   &childCount,
				LocationType: "FileSystem",
			}
			item.ImageTags = h.itemImageTags(c, showID, "series")
			if len(item.ImageTags) > 0 {
				item.SeriesPrimaryImageTag = item.ImageTags["Primary"]
			}
			item.UserData = h.getUserData(c, showID, "Series")
			seasonMap[ep.Season] = &item
		}
		*seasonMap[ep.Season].ChildCount++
	}

	items := make([]BaseItemDto, 0, len(seasonMap))
	for _, season := range seasonMap {
		items = append(items, *season)
	}

	c.JSON(http.StatusOK, ItemsResult{
		Items:            items,
		TotalRecordCount: len(items),
		StartIndex:       0,
	})
}

func (h *Handler) listEpisodesForShow(c *gin.Context, showID int64, pagination *common.PaginationParams) {
	episodesResp, err := h.tvListEpisodes.ExecuteByShowID(c.Request.Context(), showID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to list episodes"})
		return
	}

	items := make([]BaseItemDto, 0, len(episodesResp.Episodes))
	for _, ep := range episodesResp.Episodes {
		items = append(items, h.episodeToBaseItem(c, &ep, showID))
	}

	c.JSON(http.StatusOK, ItemsResult{
		Items:            items,
		TotalRecordCount: len(items),
		StartIndex:       pagination.Offset,
	})
}

func (h *Handler) listEpisodesBySeason(c *gin.Context, showID int64, seasonNum int, pagination *common.PaginationParams) {
	episodesResp, err := h.tvListEpisodes.ExecuteByShowID(c.Request.Context(), showID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to list episodes"})
		return
	}

	items := make([]BaseItemDto, 0)
	for _, ep := range episodesResp.Episodes {
		if ep.Season == seasonNum {
			items = append(items, h.episodeToBaseItem(c, &ep, showID))
		}
	}

	c.JSON(http.StatusOK, ItemsResult{
		Items:            items,
		TotalRecordCount: len(items),
		StartIndex:       pagination.Offset,
	})
}

func (h *Handler) showToBaseItem(c *gin.Context, s *apptv.TVShowDetailResponse) BaseItemDto {
	ticks := int64(0)
	item := BaseItemDto{
		Name:           s.Title,
		Id:             idStr(s.ID),
		ServerId:       "viewra",
		Type:           "Series",
		IsFolder:        true,
		RunTimeTicks:   &ticks,
		Overview:       s.Plot,
		Genres:         s.Genre,
		ProductionYear: &s.Year,
		LocationType:   "FileSystem",
	}
	if s.IMDbID != "" {
		item.ProviderIds = map[string]string{"Imdb": s.IMDbID}
	}
	if s.TMDbID > 0 {
		if item.ProviderIds == nil {
			item.ProviderIds = make(map[string]string)
		}
		item.ProviderIds["Tmdb"] = strconv.Itoa(s.TMDbID)
	}
	if s.TVDbID > 0 {
		if item.ProviderIds == nil {
			item.ProviderIds = make(map[string]string)
		}
		item.ProviderIds["Tvdb"] = strconv.Itoa(s.TVDbID)
	}
	if s.Rating > 0 {
		f := s.Rating
		item.CommunityRating = &f
	}
	if s.FirstAirDate != "" {
		item.PremiereDate = s.FirstAirDate
	}
	if s.Tagline != "" {
		item.Taglines = ifString(s.Tagline)
	}
	if s.SortTitle != "" {
		item.SortName = s.SortTitle
	}
	item.ImageTags = h.itemImageTags(c, s.ID, "series")
	item.BackdropCount = h.backdropCount(c, s.ID)
	if s.SeasonCount > 0 {
		item.ChildCount = &s.SeasonCount
	}
	item.MediaType = "Video"
	item.UserData = h.getUserData(c, s.ID, "Series")
	return item
}

func (h *Handler) episodeToBaseItem(c *gin.Context, ep *apptv.TVEpisodeResponse, showID int64) BaseItemDto {
	ticks := int64(ep.Duration) * 10000000
	item := BaseItemDto{
		Name:              ep.EpisodeTitle,
		Id:                idStr(ep.ID),
		ServerId:          "viewra",
		Type:              "Episode",
		IsFolder:           false,
		SeriesName:        ep.ShowTitle,
		IndexNumber:       &ep.Episode,
		ParentIndexNumber: &ep.Season,
		Overview:          ep.Description,
		RunTimeTicks:      &ticks,
		PremiereDate:      ep.AirDate,
		DateCreated:       fmtTime(ep.CreatedAt),
		Path:              ep.FilePath,
		Container:         ep.ContainerFormat,
		Width:             ep.Width,
		Height:            ep.Height,
		LocationType:      "FileSystem",
		MediaType:         "Video",
	}
	if showID > 0 {
		item.SeriesId = idStr(showID)
		item.SeasonId = idStr(int64(ep.Season)<<32 | 1)
	}
	if ep.IMDbID != "" {
		item.ProviderIds = map[string]string{"Imdb": ep.IMDbID}
	}
	if ep.TMDbID > 0 {
		if item.ProviderIds == nil {
			item.ProviderIds = make(map[string]string)
		}
		item.ProviderIds["Tmdb"] = strconv.FormatInt(ep.TMDbID, 10)
	}
	if ep.TVDbID > 0 {
		if item.ProviderIds == nil {
			item.ProviderIds = make(map[string]string)
		}
		item.ProviderIds["Tvdb"] = strconv.Itoa(ep.TVDbID)
	}
	if ep.Rating > 0 {
		f := ep.Rating
		item.CommunityRating = &f
	}
	if ep.OriginalTitle != "" {
		item.SortName = ep.OriginalTitle
	}
	item.SeasonName = seasonName(ep.ShowTitle, ep.Season)
	item.ImageTags = h.itemImageTags(c, ep.ID, "episode")
	item.BackdropCount = h.backdropCount(c, ep.ID)
	if len(item.ImageTags) > 0 {
		item.SeriesPrimaryImageTag = item.ImageTags["Primary"]
	}
	item.UserData = h.getUserData(c, ep.ID, "Episode")
	return item
}

func seasonName(showTitle string, season int) string {
	if season == 0 {
		return "Specials"
	}
	if showTitle != "" {
		return showTitle + " Season " + strconv.Itoa(season)
	}
	return "Season " + strconv.Itoa(season)
}
