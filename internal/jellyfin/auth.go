package jellyfin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mantonx/viewra/internal/application/auth"
)

// AuthenticateByName handles POST /Users/AuthenticateByName
// Maps Jellyfin username/password auth to ViewRA's auth service.
func (h *Handler) AuthenticateByName(c *gin.Context) {
	var req struct {
		Username string `json:"Username"`
		Pw       string `json:"Pw"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	resp, err := h.authService.Login(c.Request.Context(), &auth.LoginRequest{
		Username:  req.Username,
		Password:  req.Pw,
		UserAgent: c.Request.UserAgent(),
		IPAddress: c.ClientIP(),
	})
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	result := AuthenticationResult{
		User: &UserDto{
			Name:    resp.User.Username,
			Id:      resp.User.ID,
			ServerId: "viewra",
			HasPassword:               req.Pw != "",
			HasConfiguredPassword:     req.Pw != "",
			Configuration:             struct{}{},
			Policy:                    struct{}{},
		},
		AccessToken: resp.AccessToken,
		ServerId:    "viewra",
	}

	c.JSON(http.StatusOK, result)
}

// CreateUser handles GET /Users/New (placeholder)
func (h *Handler) CreateUser(c *gin.Context) {
	// Not implemented - ViewRA creates users via /api/auth/setup
	c.JSON(http.StatusNotImplemented, gin.H{"error": "User creation not available"})
}

// SystemInfoPublic handles GET /System/Info/Public
// Returns basic server info (no auth required in Jellyfin API).
func (h *Handler) SystemInfoPublic(c *gin.Context) {
	c.JSON(http.StatusOK, SystemInfo{
		Id:                   "viewra",
		ServerName:           "ViewRA",
		Version:              "10.8.0.0",
		ProductName:          "ViewRA Media Server",
		OperatingSystem:      "Linux",
		SystemUpdateLevel:    "Release",
		StartupWizardCompleted: true,
	})
}

// SystemPing handles GET /System/Ping
func (h *Handler) SystemPing(c *gin.Context) {
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte("ping"))
}

// SystemInfo handles GET /System/Info
func (h *Handler) SystemInfo(c *gin.Context) {
	c.JSON(http.StatusOK, SystemInfo{
		Id:                   "viewra",
		ServerName:           "ViewRA",
		Version:              "10.8.0.0",
		ProductName:          "ViewRA Media Server",
		OperatingSystem:      "Linux",
		SystemUpdateLevel:    "Release",
		StartupWizardCompleted: true,
	})
}
