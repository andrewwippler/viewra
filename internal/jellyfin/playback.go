package jellyfin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	appmedia "github.com/mantonx/viewra/internal/application/media"
	"github.com/mantonx/viewra/internal/application/progress"
)

// GetPlaybackInfo handles POST /Items/:itemId/PlaybackInfo
// Returns media sources with direct stream and HLS URLs.
func (h *Handler) GetPlaybackInfo(c *gin.Context) {
	itemID, err := parseInt64(c.Param("itemId"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	mediaResp, err := h.getMedia.Execute(c.Request.Context(), itemID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	// Parse the requested DeviceProfile (Moonfin sends this)
	// We don't do codec negotiation yet - always offer both direct and transcode
	_ = c.Request.Body

	// Extract token from context for URL construction
	token := extractJellyfinToken(c)

	// Build media source
	ms := h.buildMediaSource(c, &mediaResp, token)

	// Build media streams (audio/video/subtitle tracks)
	streams := h.buildMediaStreams(c, itemID, &mediaResp)

	ms.MediaStreams = streams

	resp := PlaybackInfoResponse{
		MediaSources: []MediaSource{ms},
		PlaySessionId: "ps_" + idStr(itemID),
	}

	c.JSON(http.StatusOK, resp)
}

// StreamVideo handles GET /Videos/:id/stream
// Redirects to ViewRA's direct stream endpoint.
func (h *Handler) StreamVideo(c *gin.Context) {
	// Let the existing ViewRA stream handler process this
	h.streamHandler.Stream(c)
}

// StreamVideoContainer handles GET /Videos/:id/stream.:container
// Redirects to ViewRA's direct stream endpoint.
func (h *Handler) StreamVideoContainer(c *gin.Context) {
	// Same as StreamVideo - container is informational for the client
	h.streamHandler.Stream(c)
}

// ServeMasterPlaylist handles GET /Videos/:id/master.m3u8
// Redirects to ViewRA's HLS master playlist.
func (h *Handler) ServeMasterPlaylist(c *gin.Context) {
	if h.transcodeH == nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "Transcoding not available"})
		return
	}
	h.transcodeH.ServeMasterPlaylist(c)
}

// ServePlaylist handles GET /Videos/:id/hls/:quality/playlist.m3u8
func (h *Handler) ServePlaylist(c *gin.Context) {
	if h.transcodeH == nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "Transcoding not available"})
		return
	}
	h.transcodeH.ServePlaylist(c)
}

// ServeHLSSegment handles GET /Videos/:id/hls/:quality/:filename
func (h *Handler) ServeHLSSegment(c *gin.Context) {
	if h.transcodeH == nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "Transcoding not available"})
		return
	}
	h.transcodeH.ServeHLSSegment(c)
}

// StartPlayback handles POST /Sessions/Playing
func (h *Handler) StartPlayback(c *gin.Context) {
	var req PlaybackStartInfo
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	mediaID, _ := parseInt64(req.ItemId)
	if mediaID > 0 && req.PositionTicks > 0 {
		userID := c.GetInt64("user_id")
		if userID == 0 {
			userID = 1
		}

		_, _ = h.progressSvc.Execute(c.Request.Context(), &progress.UpdateProgressRequest{
			MediaID:         mediaID,
			UserID:          userID,
			ProgressSeconds: ticksToSeconds(req.PositionTicks),
			DurationSeconds: 0,
		})
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ReportPlaybackProgress handles POST /Sessions/Playing/Progress
func (h *Handler) ReportPlaybackProgress(c *gin.Context) {
	var req PlaybackProgressInfo
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	mediaID, _ := parseInt64(req.ItemId)
	if mediaID > 0 && req.PositionTicks > 0 {
		userID := c.GetInt64("user_id")
		if userID == 0 {
			userID = 1
		}

		_, _ = h.progressSvc.Execute(c.Request.Context(), &progress.UpdateProgressRequest{
			MediaID:         mediaID,
			UserID:          userID,
			ProgressSeconds: ticksToSeconds(req.PositionTicks),
			DurationSeconds: 0,
		})
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// StopPlayback handles POST /Sessions/Playing/Stopped
func (h *Handler) StopPlayback(c *gin.Context) {
	var req PlaybackStopInfo
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	mediaID, _ := parseInt64(req.ItemId)
	if mediaID > 0 {
		userID := c.GetInt64("user_id")
		if userID == 0 {
			userID = 1
		}

		if req.PositionTicks > 0 {
			_, _ = h.progressSvc.Execute(c.Request.Context(), &progress.UpdateProgressRequest{
				MediaID:         mediaID,
				UserID:          userID,
				ProgressSeconds: ticksToSeconds(req.PositionTicks),
				DurationSeconds: 0,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// MarkPlayed handles POST /Users/:userId/PlayedItems/:itemId
func (h *Handler) MarkPlayed(c *gin.Context) {
	itemID, err := parseInt64(c.Param("itemId"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	userID := c.GetInt64("user_id")
	if userID == 0 {
		userID = 1
	}

	_, err = h.progressSvc.MarkWatched(c.Request.Context(), &progress.MarkWatchedRequest{
		MediaID: itemID,
		UserID:  userID,
	})
	if err != nil {
		h.logger.Error("failed to mark watched", "item_id", itemID, "error", err)
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// MarkUnplayed handles DELETE /Users/:userId/PlayedItems/:itemId
func (h *Handler) MarkUnplayed(c *gin.Context) {
	itemID, err := parseInt64(c.Param("itemId"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	userID := c.GetInt64("user_id")
	if userID == 0 {
		userID = 1
	}

	_, err = h.progressSvc.MarkUnwatched(c.Request.Context(), &progress.MarkWatchedRequest{
		MediaID: itemID,
		UserID:  userID,
	})
	if err != nil {
		h.logger.Error("failed to mark unwatched", "item_id", itemID, "error", err)
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// StreamAudio handles GET /Audio/:id/stream
func (h *Handler) StreamAudio(c *gin.Context) {
	h.streamHandler.Stream(c)
}

// StreamAudioContainer handles GET /Audio/:id/stream.:container
func (h *Handler) StreamAudioContainer(c *gin.Context) {
	h.streamHandler.Stream(c)
}

// AudioMasterPlaylist handles GET /Audio/:id/master.m3u8
func (h *Handler) AudioMasterPlaylist(c *gin.Context) {
	if h.transcodeH == nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "Transcoding not available"})
		return
	}
	h.transcodeH.ServeMasterPlaylist(c)
}

// AudioHLSSegment handles GET /Audio/:id/hls/:quality/:filename
func (h *Handler) AudioHLSSegment(c *gin.Context) {
	if h.transcodeH == nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "Transcoding not available"})
		return
	}
	h.transcodeH.ServeHLSSegment(c)
}

// StreamSubtitleVTT handles GET /Videos/:id/:mediaSourceId/Subtitles/:index/Stream.vtt
func (h *Handler) StreamSubtitleVTT(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Subtitle not found"})
}

// StreamSubtitleASS handles GET /Videos/:id/:mediaSourceId/Subtitles/:index/Stream.ass
func (h *Handler) StreamSubtitleASS(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Subtitle not found"})
}

// --- Internal helpers ---

func (h *Handler) buildMediaSource(c *gin.Context, m *appmedia.MediaResponse, token string) MediaSource {
	mediaID := idStr(m.ID)

	container := m.ContainerFormat
	if container == "" {
		container = "mp4"
	}

	directURL := "/Videos/" + mediaID + "/stream." + container + "?Static=true&mediaSourceId=" + mediaID
	if token != "" {
		directURL += "&api_key=" + token
	}

	return MediaSource{
		Id:                     mediaID,
		Name:                   m.Title,
		Type:                   "Default",
		Container:              container,
		Path:                   m.FilePath,
		RunTimeTicks:           int64(m.Duration) * 10000000,
		Size:                   m.FileSize,
		TranscodingSubProtocol: "hls",
		SupportsDirectPlay:     true,
		SupportsDirectStream:   true,
		SupportsTranscoding:    true,
		DirectStreamUrl:        directURL,
		TranscodingUrl:         "/Videos/" + mediaID + "/master.m3u8",
		Bitrate:                m.Bitrate,
	}
}

func (h *Handler) buildMediaStreams(c *gin.Context, mediaID int64, m *appmedia.MediaResponse) []MediaStream {
	streams := []MediaStream{}

	videoStream := MediaStream{
		Codec:        m.VideoCodec,
		Type:         "Video",
		Index:        0,
		Width:        m.Width,
		Height:       m.Height,
		IsInterlaced: false,
		BitRate:      m.Bitrate,
	}
	streams = append(streams, videoStream)

	if h.getTracks != nil {
		tracks, err := h.getTracks.Execute(c.Request.Context(), mediaID)
		if err == nil && tracks != nil {
			streamIdx := 1
			for _, a := range tracks.AudioTracks {
				chLayout := a.ChannelLayout
				sampleRate := a.SampleRate
				bitRate := a.BitRate
				stream := MediaStream{
					Codec:         a.Codec,
					Type:          "Audio",
					Index:         streamIdx,
					Language:      a.Language,
					Title:         a.Title,
					DisplayTitle:  a.Title,
					ChannelLayout: chLayout,
					SampleRate:    sampleRate,
					BitRate:       int64(bitRate),
					IsDefault:     a.IsDefault,
				}
				streams = append(streams, stream)
				streamIdx++
			}
			for _, s := range tracks.SubtitleTracks {
				stream := MediaStream{
					Codec:        s.Codec,
					Type:         "Subtitle",
					Index:        streamIdx,
					Language:     s.Language,
					Title:        s.Title,
					DisplayTitle: s.Title,
					IsDefault:    s.IsDefault,
					IsForced:     s.IsForced,
				}
				streams = append(streams, stream)
				streamIdx++
			}
			return streams
		}
	}

	audioStream := MediaStream{
		Codec:     m.AudioCodec,
		Type:      "Audio",
		Index:     1,
		IsDefault: true,
	}
	streams = append(streams, audioStream)

	return streams
}
