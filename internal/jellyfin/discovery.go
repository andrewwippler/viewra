package jellyfin

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// QuickConnectEnabled handles GET /QuickConnect/Enabled
func (h *Handler) QuickConnectEnabled(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"Enabled": false,
	})
}

// QuickConnectInitiate handles GET /QuickConnect/Initiate
func (h *Handler) QuickConnectInitiate(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"Code":   "",
		"Secret": "",
	})
}

// QuickConnectConnect handles GET /QuickConnect/Connect
func (h *Handler) QuickConnectConnect(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"Result": "Failure",
	})
}

// GetCurrentUser handles GET /Users/Me
func (h *Handler) GetCurrentUser(c *gin.Context) {
	userID := c.GetInt64("user_id")
	claims := c.GetString("claims")

	_ = claims

	c.JSON(http.StatusOK, UserDto{
		Name:                "User",
		Id:                  idStr(userID),
		ServerId:            "viewra",
		HasPassword:         true,
		HasConfiguredPassword: true,
		Configuration:       struct{}{},
		Policy:              struct{}{},
	})
}

// BrandingConfiguration handles GET /Branding/Configuration
func (h *Handler) BrandingConfiguration(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}

// SystemConfiguration handles GET /System/Configuration
func (h *Handler) SystemConfiguration(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}

// DisplayPreferences handles GET /DisplayPreferences/usersettings
func (h *Handler) DisplayPreferences(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"Id": "usersettings",
	})
}
