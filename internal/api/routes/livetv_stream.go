package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/mantonx/viewra/internal/api/handlers"
)

func RegisterLiveTvStreamRoutes(rg *gin.RouterGroup, handler *handlers.LiveTvStreamHandler) {
	stream := rg.Group("/livetv/:libraryId/channels/:channelId")
	stream.GET("/hls/playlist.m3u8", handler.GetPlaylist)
	stream.GET("/hls/live/playlist.m3u8", handler.GetPlaylist)
	stream.GET("/hls/live/:filename", handler.GetSegment)
	stream.GET("/hls/:filename", handler.GetSegment)
	stream.GET("/status", handler.StreamStatus)
	stream.POST("/pause", handler.Pause)
	stream.POST("/resume", handler.Resume)
	stream.POST("/stop", handler.Stop)
}