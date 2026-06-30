package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/mantonx/viewra/internal/api/handlers"
)

// RegisterMediaRoutes registers all media-related routes
func RegisterMediaRoutes(rg *gin.RouterGroup, handler *handlers.MediaHandler) {
	media := rg.Group("/media")
	media.GET("", handler.List)
	media.GET("/:id", handler.Get)
	media.DELETE("/:id", handler.Delete)
	media.GET("/:id/stream-info", handler.GetStreamInfo)
	media.GET("/:id/tracks", handler.GetTracks)
}
