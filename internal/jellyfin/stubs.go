package jellyfin

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// GetSimilarItems handles GET /Items/:itemId/Similar
func (h *Handler) GetSimilarItems(c *gin.Context) {
	c.JSON(http.StatusOK, ItemsResult{
		Items:            []BaseItemDto{},
		TotalRecordCount: 0,
		StartIndex:       0,
	})
}

// GetAncestors handles GET /Items/:itemId/Ancestors
func (h *Handler) GetAncestors(c *gin.Context) {
	c.JSON(http.StatusOK, []BaseItemDto{})
}

// GetThemeMedia handles GET /Items/:itemId/ThemeMedia
func (h *Handler) GetThemeMedia(c *gin.Context) {
	c.JSON(http.StatusOK, ItemsResult{
		Items:            []BaseItemDto{},
		TotalRecordCount: 0,
		StartIndex:       0,
	})
}

// GetPlugins handles GET /Plugins
func (h *Handler) GetPlugins(c *gin.Context) {
	c.JSON(http.StatusOK, []struct{}{})
}

// GetMediaSegments handles GET /MediaSegments/:itemId
func (h *Handler) GetMediaSegments(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}

// LiveTvChannels handles GET /LiveTv/Channels
func (h *Handler) LiveTvChannels(c *gin.Context) {
	c.JSON(http.StatusOK, ItemsResult{
		Items:            []BaseItemDto{},
		TotalRecordCount: 0,
		StartIndex:       0,
	})
}

// LiveTvPrograms handles GET /LiveTv/Programs
func (h *Handler) LiveTvPrograms(c *gin.Context) {
	c.JSON(http.StatusOK, ItemsResult{
		Items:            []BaseItemDto{},
		TotalRecordCount: 0,
		StartIndex:       0,
	})
}

// LiveTvEmpty handles GET /LiveTv/Recordings, /LiveTv/Timers, /LiveTv/SeriesTimers
func (h *Handler) LiveTvEmpty(c *gin.Context) {
	c.JSON(http.StatusOK, ItemsResult{
		Items:            []BaseItemDto{},
		TotalRecordCount: 0,
		StartIndex:       0,
	})
}

// LiveTvGuideInfo handles GET /LiveTv/GuideInfo
func (h *Handler) LiveTvGuideInfo(c *gin.Context) {
	now := time.Now()
	c.JSON(http.StatusOK, gin.H{
		"StartDate": now.Format(time.RFC3339),
		"EndDate":   now.AddDate(0, 0, 7).Format(time.RFC3339),
	})
}

// LiveTvRecommendedPrograms handles GET /LiveTv/RecommendedPrograms
func (h *Handler) LiveTvRecommendedPrograms(c *gin.Context) {
	c.JSON(http.StatusOK, ItemsResult{
		Items:            []BaseItemDto{},
		TotalRecordCount: 0,
		StartIndex:       0,
	})
}

// GetSessions handles GET /Sessions
func (h *Handler) GetSessions(c *gin.Context) {
	c.JSON(http.StatusOK, []struct{}{})
}

// GetGenres handles GET /Genres
func (h *Handler) GetGenres(c *gin.Context) {
	c.JSON(http.StatusOK, ItemsResult{
		Items:            []BaseItemDto{},
		TotalRecordCount: 0,
		StartIndex:       0,
	})
}

// GetPersons handles GET /Persons
func (h *Handler) GetPersons(c *gin.Context) {
	c.JSON(http.StatusOK, ItemsResult{
		Items:            []BaseItemDto{},
		TotalRecordCount: 0,
		StartIndex:       0,
	})
}

// GetSpecialFeatures handles GET /Items/:itemId/SpecialFeatures
func (h *Handler) GetSpecialFeatures(c *gin.Context) {
	c.JSON(http.StatusOK, []BaseItemDto{})
}

// GetAdditionalParts handles GET /Items/:itemId/AdditionalParts
func (h *Handler) GetAdditionalParts(c *gin.Context) {
	c.JSON(http.StatusOK, []BaseItemDto{})
}

// GetLocalTrailers handles GET /Items/:itemId/LocalTrailers
func (h *Handler) GetLocalTrailers(c *gin.Context) {
	c.JSON(http.StatusOK, []BaseItemDto{})
}

// GetIntros handles GET /Items/:itemId/Intros
func (h *Handler) GetIntros(c *gin.Context) {
	c.JSON(http.StatusOK, []BaseItemDto{})
}

// GetSuggestions handles GET /Users/:userId/Suggestions
func (h *Handler) GetSuggestions(c *gin.Context) {
	limit, _ := parseInt64(c.DefaultQuery("Limit", "20"))

	mediaResp, err := h.listMedia.ExecuteAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusOK, ItemsResult{
			Items:            []BaseItemDto{},
			TotalRecordCount: 0,
			StartIndex:       0,
		})
		return
	}

	items := make([]BaseItemDto, 0, limit)
	for i := len(mediaResp.Media) - 1; i >= 0 && int64(len(items)) < limit; i-- {
		m := mediaResp.Media[i]
		items = append(items, h.mediaToBaseItem(c, &m))
	}

	c.JSON(http.StatusOK, ItemsResult{
		Items:            items,
		TotalRecordCount: len(items),
		StartIndex:       0,
	})
}

// GetNotificationSummary handles GET /Notifications/Summary
func (h *Handler) GetNotificationSummary(c *gin.Context) {
	c.JSON(http.StatusOK, NotificationSummary{
		Unread:                 0,
		MaxUnreadNotificationLevel: "Info",
	})
}

// GetNotificationTypes handles GET /Notifications/Types
func (h *Handler) GetNotificationTypes(c *gin.Context) {
	c.JSON(http.StatusOK, []struct{}{})
}

// GetSystemLogs handles GET /System/Logs
func (h *Handler) GetSystemLogs(c *gin.Context) {
	c.JSON(http.StatusOK, []struct{}{})
}

// GetPackages handles GET /Packages
func (h *Handler) GetPackages(c *gin.Context) {
	c.JSON(http.StatusOK, []struct{}{})
}
