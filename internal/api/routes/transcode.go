package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/mantonx/viewra/internal/api/handlers"
	"github.com/mantonx/viewra/internal/api/middleware"
)

// RegisterTranscodeRoutes registers all transcode-related routes
func RegisterTranscodeRoutes(router *gin.RouterGroup, handler *handlers.TranscodeHandler) {
	// Skip route registration if handler is nil
	if handler == nil {
		return
	}

	// Note: FFmpeg log routes are registered separately via RegisterFFmpegLogRoutes

	// POST /api/media/:id/transcode/:quality - Create transcode job
	router.POST("/media/:id/transcode/:quality", handler.CreateTranscodeJob)

	// GET /api/media/:id/transcode/:quality - Get transcode job status
	router.GET("/media/:id/transcode/:quality", handler.GetTranscodeStatus)

	// HLS streaming routes need broader CORS for cross-origin segment loading
	hls := router.Group("/media/:id/hls")
	hls.Use(middleware.StreamingCORS())

	// GET /api/media/:id/hls/master.m3u8 - Serve HLS master playlist with all quality variants
	hls.GET("/master.m3u8", handler.ServeMasterPlaylist)

	// Subtitle routes for HLS subtitle support (WebVTT)
	// GET /api/media/:id/hls/subtitle/:trackIndex/subtitles.vtt - Serve subtitle WebVTT file
	hls.GET("/subtitle/:trackIndex/subtitles.vtt", handler.ServeSubtitle)

	// GET /api/media/:id/hls/heartbeat - Keep transcode session alive (pause heartbeat)
	hls.GET("/heartbeat", handler.ServeHeartbeat)

	// GET /api/media/:id/hls/:quality/playlist.m3u8 - Serve HLS playlist (with on-demand transcoding)
	hls.GET("/:quality/playlist.m3u8", handler.ServePlaylist)

	// GET /api/media/:id/hls/:quality/:filename - Serve HLS segment files (MPEG-TS segments)
	hls.GET("/:quality/:filename", handler.ServeHLSSegment)

	// GET /api/transcode/queue - Get queue statistics
	router.GET("/transcode/queue", handler.GetQueueStats)

	// POST /api/media/:id/transcode/:quality/cancel - Cancel transcode job
	router.POST("/media/:id/transcode/:quality/cancel", handler.CancelTranscodeJob)

	// GET /api/transcode/disk-usage - Get disk usage statistics
	router.GET("/transcode/disk-usage", handler.GetDiskUsage)

	// POST /api/transcode/cleanup - Cleanup transcode files
	router.POST("/transcode/cleanup", handler.CleanupTranscodes)
}

// RegisterFFmpegLogRoutes registers routes for FFmpeg log access and debugging.
func RegisterFFmpegLogRoutes(router *gin.RouterGroup, handler *handlers.FFmpegLogsHandler) {
	// Skip route registration if handler is nil
	if handler == nil {
		return
	}

	// GET /api/media/:id/ffmpeg-logs - List FFmpeg logs for media
	router.GET("/media/:id/ffmpeg-logs", handler.ListLogs)

	// GET /api/media/:id/ffmpeg-logs/:session_id - Get specific log content
	router.GET("/media/:id/ffmpeg-logs/:session_id", handler.GetLog)

	// GET /api/media/:id/ffmpeg-logs/:session_id/info - Get log metadata
	router.GET("/media/:id/ffmpeg-logs/:session_id/info", handler.GetLogInfo)

	// GET /api/media/:id/ffmpeg-logs/:session_id/stream - Stream log in real-time (SSE)
	// SSE needs streaming CORS for cross-origin event source connections
	router.GET("/media/:id/ffmpeg-logs/:session_id/stream", middleware.StreamingCORS(), handler.StreamLog)

	// DELETE /api/media/:id/ffmpeg-logs/:session_id - Delete specific log
	router.DELETE("/media/:id/ffmpeg-logs/:session_id", handler.DeleteLog)

	// GET /api/transcode/active-sessions - List active transcode sessions
	router.GET("/transcode/active-sessions", handler.ListActiveSessions)
}
