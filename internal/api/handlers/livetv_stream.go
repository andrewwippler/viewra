package handlers

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mantonx/viewra/internal/infrastructure/transcoding/session"
)

// LiveTvStreamHandler handles live TV HLS streaming and DVR controls.
type LiveTvStreamHandler struct {
	sessionManager   *session.Manager
	getChannelByID   func(ctx context.Context, libraryID, channelID int64) (string, error)
}

// NewLiveTvStreamHandler creates a new live TV stream handler.
func NewLiveTvStreamHandler(sessionManager *session.Manager, getChannelByID func(ctx context.Context, libraryID, channelID int64) (string, error)) *LiveTvStreamHandler {
	return &LiveTvStreamHandler{
		sessionManager: sessionManager,
		getChannelByID: getChannelByID,
	}
}

// GetPlaylist returns the HLS playlist for a channel, auto-creating the transcode session if needed.
func (h *LiveTvStreamHandler) GetPlaylist(c *gin.Context) {
	libraryID, err := parseID(c.Param("libraryId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_LIBRARY_ID", "Invalid library ID")
		return
	}

	channelID, err := parseID(c.Param("channelId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_CHANNEL_ID", "Invalid channel ID")
		return
	}

	// Try to get existing session output path
	outputPath, err := h.sessionManager.GetLiveStreamOutputPath(libraryID, channelID)
	if err != nil {
		// No active session - create one
		if h.getChannelByID == nil {
			respondError(c, http.StatusServiceUnavailable, "SESSION_NOT_FOUND", "No active stream and cannot create one")
			return
		}

		// Get channel stream URL from database
		streamURL, err := h.getChannelByID(c.Request.Context(), libraryID, channelID)
		if err != nil {
			respondError(c, http.StatusNotFound, "CHANNEL_NOT_FOUND", "Channel not found")
			return
		}

		// Create new live stream session
		_, err = h.sessionManager.GetOrCreateLiveStreamSession(session.LiveStreamSessionParams{
			LibraryID:      libraryID,
			ChannelID:      channelID,
			InputURL:       streamURL,
			MaxDVRSegments: 1800,
		})
		if err != nil {
			respondError(c, http.StatusInternalServerError, "SESSION_CREATE_FAILED", "Failed to start stream: "+err.Error())
			return
		}

		// Get output path after creation
		outputPath, err = h.sessionManager.GetLiveStreamOutputPath(libraryID, channelID)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "SESSION_ERROR", "Stream started but output not available")
			return
		}
	}

	playlistPath := outputPath + "/playlist.m3u8"

	// Wait for playlist to exist (FFmpeg needs time to initialize)
	for i := 0; i < 60; i++ {
		if _, err := os.Stat(playlistPath); err == nil {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}

	// Check if playlist exists now
	if _, err := os.Stat(playlistPath); os.IsNotExist(err) {
		respondError(c, http.StatusServiceUnavailable, "PLAYLIST_NOT_READY", "Stream is initializing, please retry")
		return
	}

	// Serve the playlist
	c.Header("Content-Type", "application/vnd.apple.mpegurl")
	c.File(playlistPath)
}

// GetSegment returns an HLS segment file.
func (h *LiveTvStreamHandler) GetSegment(c *gin.Context) {
	libraryID, err := parseID(c.Param("libraryId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_LIBRARY_ID", "Invalid library ID")
		return
	}

	channelID, err := parseID(c.Param("channelId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_CHANNEL_ID", "Invalid channel ID")
		return
	}

	filename := c.Param("filename")
	if filename == "" || !strings.HasSuffix(filename, ".ts") {
		respondError(c, http.StatusBadRequest, "INVALID_FILENAME", "Invalid segment filename")
		return
	}

	// Get session output path
	outputPath, err := h.sessionManager.GetLiveStreamOutputPath(libraryID, channelID)
	if err != nil {
		respondError(c, http.StatusNotFound, "SESSION_NOT_FOUND", "No active stream for this channel")
		return
	}

	segmentPath := outputPath + "/" + filename

	// Serve the segment file
	c.File(segmentPath)
}

// PauseRequest represents the pause request body.
type PauseRequest struct {
	Position float64 `json:"position"`
}

// PauseResponse represents the pause response.
type PauseResponse struct {
	Paused          bool    `json:"paused"`
	Position        float64 `json:"position"`
	RetainedSegments int    `json:"retained_segments"`
}

// Pause pauses the live stream, enabling DVR.
func (h *LiveTvStreamHandler) Pause(c *gin.Context) {
	libraryID, err := parseID(c.Param("libraryId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_LIBRARY_ID", "Invalid library ID")
		return
	}

	channelID, err := parseID(c.Param("channelId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_CHANNEL_ID", "Invalid channel ID")
		return
	}

	// Parse request body
	var req PauseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Position is optional, default to 0
		req.Position = 0
	}

	// Get session
	sess, err := h.sessionManager.GetLiveStreamSession(libraryID, channelID)
	if err != nil {
		respondError(c, http.StatusNotFound, "SESSION_NOT_FOUND", "No active stream for this channel")
		return
	}

	// Check if another session is already paused (409 Conflict)
	if err := h.sessionManager.CheckPauseConflict(sess); err != nil {
		respondError(c, http.StatusConflict, "ANOTHER_PAUSED", err.Error())
		return
	}

	// Pause the session
	position, err := sess.Pause(req.Position)
	if err != nil {
		if strings.Contains(err.Error(), "ended") {
			respondError(c, http.StatusGone, "STREAM_ENDED", "DVR limit reached, stream has ended")
			return
		}
		respondError(c, http.StatusInternalServerError, "PAUSE_FAILED", err.Error())
		return
	}

	// Register this as the paused session
	h.sessionManager.RegisterPausedSession(sess)

	c.JSON(http.StatusOK, PauseResponse{
		Paused:           true,
		Position:         position,
		RetainedSegments: sess.GetRetainedSegments(),
	})
}

// ResumeResponse represents the resume response.
type ResumeResponse struct {
	Resumed  bool    `json:"resumed"`
	Position float64 `json:"position"`
}

// Resume resumes the live stream from the paused position.
func (h *LiveTvStreamHandler) Resume(c *gin.Context) {
	libraryID, err := parseID(c.Param("libraryId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_LIBRARY_ID", "Invalid library ID")
		return
	}

	channelID, err := parseID(c.Param("channelId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_CHANNEL_ID", "Invalid channel ID")
		return
	}

	// Get session
	sess, err := h.sessionManager.GetLiveStreamSession(libraryID, channelID)
	if err != nil {
		respondError(c, http.StatusNotFound, "SESSION_NOT_FOUND", "No active stream for this channel")
		return
	}

	// Resume the session
	position, err := sess.Resume()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "RESUME_FAILED", err.Error())
		return
	}

	// Clear the paused session reference
	h.sessionManager.ClearPausedSession(sess)

	c.JSON(http.StatusOK, ResumeResponse{
		Resumed:  true,
		Position: position,
	})
}

// StopResponse represents the stop response.
type StopResponse struct {
	Stopped bool `json:"stopped"`
}

// Stop stops the live stream.
func (h *LiveTvStreamHandler) Stop(c *gin.Context) {
	libraryID, err := parseID(c.Param("libraryId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_LIBRARY_ID", "Invalid library ID")
		return
	}

	channelID, err := parseID(c.Param("channelId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_CHANNEL_ID", "Invalid channel ID")
		return
	}

	// Stop the session
	if err := h.sessionManager.StopLiveStreamSession(libraryID, channelID); err != nil {
		respondError(c, http.StatusNotFound, "SESSION_NOT_FOUND", "No active stream for this channel")
		return
	}

	c.JSON(http.StatusOK, StopResponse{
		Stopped: true,
	})
}

// StreamStatusResponse represents the stream status response.
type StreamStatusResponse struct {
	Active           bool    `json:"active"`
	State            string  `json:"state"`
	Position         float64 `json:"position,omitempty"`
	RetainedSegments int     `json:"retained_segments,omitempty"`
}

// StreamStatus returns the current stream status.
func (h *LiveTvStreamHandler) StreamStatus(c *gin.Context) {
	libraryID, err := parseID(c.Param("libraryId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_LIBRARY_ID", "Invalid library ID")
		return
	}

	channelID, err := parseID(c.Param("channelId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_CHANNEL_ID", "Invalid channel ID")
		return
	}

	// Get session status
	status, err := h.sessionManager.GetLiveStreamStatus(libraryID, channelID)
	if err != nil {
		c.JSON(http.StatusOK, StreamStatusResponse{
			Active: false,
			State:  "inactive",
		})
		return
	}

	stateStr := "playing"
	switch status.State {
	case session.LiveStatePaused:
		stateStr = "paused"
	case session.LiveStateEnded:
		stateStr = "ended"
	}

	c.JSON(http.StatusOK, StreamStatusResponse{
		Active:           true,
		State:            stateStr,
		Position:         status.Position,
		RetainedSegments: status.RetainedSegments,
	})
}