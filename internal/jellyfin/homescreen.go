package jellyfin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mantonx/viewra/internal/domain/home"
)

// HomeScreenMeta handles GET /HomeScreen/Meta
func (h *Handler) HomeScreenMeta(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}

// HomeScreenSections handles GET /HomeScreen/Sections
func (h *Handler) HomeScreenSections(c *gin.Context) {
	if h.homeService != nil {
		userID := c.GetInt64("user_id")
		resp, err := h.homeService.GetHome(c.Request.Context(), &home.HomeRequest{
			UserID: idStr(userID),
		})
		if err == nil && resp != nil {
			sections := make([]gin.H, 0, len(resp.Sections))
			for range resp.Sections {
				sections = append(sections, gin.H{
					"Name": "Section",
					"Type": "LibraryView",
				})
			}
			c.JSON(http.StatusOK, gin.H{"Items": sections})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"Items": []struct{}{}})
}

// HomeScreenSection handles GET /HomeScreen/Section/:sectionType
func (h *Handler) HomeScreenSection(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"Items": []struct{}{}})
}
