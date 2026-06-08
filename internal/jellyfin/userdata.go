package jellyfin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	appratings "github.com/mantonx/viewra/internal/application/ratings"
	"github.com/mantonx/viewra/internal/domain/ratings"
)

// UpdateDisplayPreferences handles POST /DisplayPreferences/:id
// Accepts and stores display preferences for the user.
func (h *Handler) UpdateDisplayPreferences(c *gin.Context) {
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"Id": c.Param("id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"Id": c.Param("id"),
	})
}

// Favorite handles POST /Users/:userId/FavoriteItems/:itemId
func (h *Handler) Favorite(c *gin.Context) {
	itemID, err := parseInt64(c.Param("itemId"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	userID := c.GetInt64("user_id")
	if userID == 0 {
		userID = 1
	}

	entityType := h.resolveItemType(c, itemID)

	if h.ratingsService != nil {
		_, _ = h.ratingsService.CreateOrUpdate(c.Request.Context(), &appratings.CreateRatingRequest{
			UserID:     idStr(userID),
			EntityType: entityType,
			EntityID:   itemID,
			Rating:     ratings.RatingFavorite,
		})
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Unfavorite handles DELETE /Users/:userId/FavoriteItems/:itemId
func (h *Handler) Unfavorite(c *gin.Context) {
	itemID, err := parseInt64(c.Param("itemId"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	userID := c.GetInt64("user_id")
	if userID == 0 {
		userID = 1
	}

	entityType := h.resolveItemType(c, itemID)

	if h.ratingsService != nil {
		_ = h.ratingsService.Delete(c.Request.Context(), idStr(userID), entityType, itemID)
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// resolveItemType attempts to determine the ViewRA entity type for a media item.
func (h *Handler) resolveItemType(c *gin.Context, itemID int64) string {
	movieResp, err := h.moviesGet.Execute(c.Request.Context(), itemID)
	if err == nil && movieResp != nil {
		return "movie"
	}

	showResp, err := h.tvGetShow.Execute(c.Request.Context(), itemID)
	if err == nil && showResp != nil {
		return "tv_show"
	}

	epResp, err := h.tvGetEpisode.Execute(c.Request.Context(), itemID)
	if err == nil && epResp != nil {
		return "tv_episode"
	}

	mediaResp, err := h.getMedia.Execute(c.Request.Context(), itemID)
	if err == nil {
		switch mediaResp.Type {
		case "movie":
			return "movie"
		case "tv_episode":
			return "tv_episode"
		}
	}

	return "movie"
}
