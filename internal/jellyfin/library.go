package jellyfin

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mantonx/viewra/internal/domain/common"
)

// GetViews handles GET /Users/:userId/Views
func (h *Handler) GetViews(c *gin.Context) {
	libs, err := h.libraryService.List(c.Request.Context())
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to list libraries"})
		return
	}

	items := make([]BaseItemDto, 0, len(libs.Libraries))
	for _, lib := range libs.Libraries {
		item := BaseItemDto{
			Name:         lib.Name,
			Id:           idStr(lib.ID),
			ServerId:     "viewra",
			Type:         "CollectionFolder",
			CollectionType: mapViewraLibTypeToJellyfin(lib.Type),
			IsFolder:      true,
			LocationType:  "FileSystem",
		}
		item.ImageTags = h.itemImageTags(c, lib.ID, "library")
		items = append(items, item)
	}

	c.JSON(http.StatusOK, ItemsResult{
		Items:            items,
		TotalRecordCount: len(items),
		StartIndex:       0,
	})
}

// GetUser handles GET /Users/:userId
func (h *Handler) GetUser(c *gin.Context) {
	c.JSON(http.StatusOK, UserDto{
		Name:                  "User",
		Id:                    c.Param("userId"),
		ServerId:              "viewra",
		HasPassword:           true,
		HasConfiguredPassword: true,
		Configuration:         struct{}{},
		Policy:                struct{}{},
	})
}

// PublicUsers handles GET /Users/Public
func (h *Handler) PublicUsers(c *gin.Context) {
	c.JSON(http.StatusOK, []UserDto{})
}

// AuthenticateWithQuickConnect handles POST /Users/AuthenticateWithQuickConnect
func (h *Handler) AuthenticateWithQuickConnect(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "QuickConnect not available"})
}

// ReportCapabilities handles POST /Sessions/Capabilities/Full
func (h *Handler) ReportCapabilities(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) listLibraryItems(c *gin.Context, libraryID int64, types []string, sortBy, sortOrder string, pagination *common.PaginationParams) {
	hasMovie := len(types) == 0 || types[0] == "" || contains(types, "Movie")
	hasSeries := contains(types, "Series")
	hasEpisode := contains(types, "Episode")

	items := make([]BaseItemDto, 0)

	if hasMovie {
		moviesResp, err := h.moviesList.Execute(c.Request.Context(), libraryID)
		if err == nil {
			for _, m := range moviesResp.Movies {
				items = append(items, h.movieToBaseItem(c, &m))
			}
		}
	}

	if hasSeries {
		showsResp, err := h.tvListShows.Execute(c.Request.Context(), libraryID)
		if err == nil {
			for _, s := range showsResp.Shows {
				ticks := int64(0)
				item := BaseItemDto{
					Name:         s.Title,
					Id:           idStr(s.ID),
					ServerId:     "viewra",
					Type:         "Series",
					IsFolder:      true,
					RunTimeTicks: &ticks,
					Overview:     s.Plot,
					Genres:       s.Genre,
					ProductionYear: &s.Year,
					LocationType: "FileSystem",
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
		}
	}

	if hasEpisode {
		episodesResp, err := h.tvListEpisodes.ExecuteByLibrary(c.Request.Context(), libraryID)
		if err == nil {
			for _, ep := range episodesResp.Episodes {
				items = append(items, h.episodeToBaseItem(c, &ep, 0))
			}
		}
	}

	_ = sortBy
	_ = sortOrder

	c.JSON(http.StatusOK, ItemsResult{
		Items:            items,
		TotalRecordCount: len(items),
		StartIndex:       pagination.Offset,
	})
}
