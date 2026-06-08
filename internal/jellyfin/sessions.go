package jellyfin

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Logout handles POST /Sessions/Logout
// Logs out the current user by invalidating their access token.
func (h *Handler) Logout(c *gin.Context) {
	token := extractJellyfinToken(c)
	if token == "" {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	// Try to extract refresh token from header or body
	var req struct {
		RefreshToken string `json:"RefreshToken"`
	}
	if err := c.ShouldBindJSON(&req); err == nil && req.RefreshToken != "" {
		_ = h.authService.Logout(c.Request.Context(), req.RefreshToken)
	} else {
		// If no refresh token, attempt logout via access token context
		_ = h.authService.Logout(c.Request.Context(), token)
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
