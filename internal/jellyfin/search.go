package jellyfin

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mantonx/viewra/internal/domain/search"
)

// GetSearchHints handles GET /Search/Hints
// Returns search autocomplete results in Jellyfin SearchHint format.
func (h *Handler) GetSearchHints(c *gin.Context) {
	query := c.Query("SearchTerm")
	limit, _ := strconv.Atoi(c.DefaultQuery("Limit", "20"))
	mediaTypes := c.Query("IncludeItemTypes")

	if query == "" {
		c.JSON(http.StatusOK, SearchHintResult{
			SearchHints:      []SearchHint{},
			TotalRecordCount: 0,
		})
		return
	}

	var typeFilter []string
	if mediaTypes != "" {
		typeFilter = []string{mediaTypes}
	}

	searchResp, err := h.searchService.Search(c.Request.Context(), &search.Request{
		Query:      query,
		MediaTypes: typeFilter,
		Limit:      limit,
	})
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}

	hints := make([]SearchHint, 0, len(searchResp.Results))
	for _, r := range searchResp.Results {
		jellyfinType := mapSearchTypeToJellyfin(r.MediaType)
		mediaType := "Video"
		if r.MediaType == "music_track" || r.MediaType == "music_artist" || r.MediaType == "music_album" {
			mediaType = "Audio"
		}

		hint := SearchHint{
			ItemId:    idStr(r.ID),
			Id:        idStr(r.ID),
			Name:      r.Title,
			Type:      jellyfinType,
			MediaType: mediaType,
			ProductionYear: &r.Year,
		}

		// Look up full item to get image tags
		mediaResp, err := h.getMedia.Execute(c.Request.Context(), r.ID)
		if err == nil {
			imgTags := h.itemImageTags(c, r.ID, r.MediaType)
			if primary, ok := imgTags["Primary"]; ok {
				hint.PrimaryImageTag = primary
			}
			ticks := int64(mediaResp.Duration) * 10000000
			hint.RunTimeTicks = &ticks
		}

		hints = append(hints, hint)
	}

	c.JSON(http.StatusOK, SearchHintResult{
		SearchHints:      hints,
		TotalRecordCount: len(hints),
	})
}
