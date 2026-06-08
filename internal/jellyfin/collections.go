package jellyfin

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetCollections handles GET /Collections
// Returns all libraries as collections.
func (h *Handler) GetCollections(c *gin.Context) {
	libs, err := h.libraryService.List(c.Request.Context())
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to list libraries"})
		return
	}

	items := make([]BaseItemDto, 0, len(libs.Libraries))
	for _, lib := range libs.Libraries {
		item := BaseItemDto{
			Name:          lib.Name,
			Id:            idStr(lib.ID),
			ServerId:      "viewra",
			Type:          "CollectionFolder",
			CollectionType: mapViewraLibTypeToJellyfin(lib.Type),
			IsFolder:       true,
			LocationType:   "FileSystem",
		}
		item.ImageTags = h.itemImageTags(c, lib.ID, "library")
		items = append(items, item)
	}

	c.JSON(http.StatusOK, items)
}

// GetCollectionItems handles GET /Collections/:id/Items
// Returns items within a collection (library).
func (h *Handler) GetCollectionItems(c *gin.Context) {
	collectionID, err := parseInt64(c.Param("id"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid collection ID"})
		return
	}

	startIndex, _ := strconv.Atoi(c.DefaultQuery("StartIndex", "0"))
	_, _ = strconv.Atoi(c.DefaultQuery("Limit", "100"))

	includeTypes := c.Query("IncludeItemTypes")

	items := make([]BaseItemDto, 0)

	movieResp, err := h.moviesList.Execute(c.Request.Context(), collectionID)
	if err == nil {
		for _, m := range movieResp.Movies {
			if includeTypes != "" && includeTypes != "Movie" {
				continue
			}
			items = append(items, h.movieToBaseItem(c, &m))
		}
	}

	showResp, err := h.tvListShows.Execute(c.Request.Context(), collectionID)
	if err == nil {
		for _, s := range showResp.Shows {
			if includeTypes != "" && includeTypes != "Series" {
				continue
			}
			showDetail, detailErr := h.tvGetShow.Execute(c.Request.Context(), s.ID)
			if detailErr == nil {
				items = append(items, h.showToBaseItem(c, showDetail))
			}
		}
	}

	c.JSON(http.StatusOK, ItemsResult{
		Items:            items,
		TotalRecordCount: len(items),
		StartIndex:       startIndex,
	})
}
