package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/mantonx/viewra/internal/api/handlers"
	"github.com/mantonx/viewra/internal/api/middleware"
)

func RegisterLiveTvStreamRoutes(rg *gin.RouterGroup, handler *handlers.LiveTvStreamHandler) {
	// HLS streaming routes need broader CORS for cross-origin segment loading
	stream := rg.Group("/livetv/:libraryId/channels/:channelId/hls")
	stream.Use(middleware.StreamingCORS())

	stream.GET("/playlist.m3u8", handler.GetPlaylist)
	stream.GET("/live/playlist.m3u8", handler.GetPlaylist)
	stream.GET("/live/:filename", handler.GetSegment)
	stream.GET("/:filename", handler.GetSegment)

	// Non-streaming routes use standard CORS
	status := rg.Group("/livetv/:libraryId/channels/:channelId")
	status.GET("/status", handler.StreamStatus)
	status.POST("/pause", handler.Pause)
	status.POST("/resume", handler.Resume)
	status.POST("/stop", handler.Stop)
}