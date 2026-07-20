package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mantonx/viewra/internal/application/transcode"
	"github.com/mantonx/viewra/internal/infrastructure/subtitles"
	"github.com/mantonx/viewra/internal/infrastructure/transcoding/segment"
)

// ServePlaylist serves the HLS playlist file for a media item with on-demand segment generation.
//
// @Summary Serve HLS playlist (instant manifest generation)
// @Description Serves the HLS playlist (.m3u8) file for adaptive streaming. Generates complete manifest instantly
// @Description from segment 0. Segments are created on-demand as the player requests them. Compatible videos redirect to direct stream.
// @Tags transcode
// @Produce application/vnd.apple.mpegurl,application/json
// @Param id path int true "Media ID"
// @Param quality path string true "Quality level (360p, 720p, 1080p, 4k)"
// @Success 200 {file} file "HLS playlist file - segments generated on-demand"
// @Success 302 "Redirect to direct stream (for compatible files)"
// @Failure 400 {object} handlers.APIError
// @Failure 404 {object} handlers.APIError
// @Failure 500 {object} handlers.APIError
// @Router /api/media/{id}/hls/{quality}/playlist.m3u8 [get]
func (h *TranscodeHandler) ServePlaylist(c *gin.Context) {
	mediaIDStr := c.Param("id")
	quality := c.Param("quality")

	mediaID, err := parseID(mediaIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid media ID")
		return
	}

	// Parse optional start position query parameter for seeking
	startPosition := 0.0
	if startStr := c.Query("start"); startStr != "" {
		if start, err := parseFloat(startStr); err == nil {
			startPosition = start
		}
	}

	// Parse optional audio track index for multi-audio selection
	// -1 means use default (first audio track), >= 0 is the FFmpeg stream index
	audioTrackIndex := -1
	if audioTrackStr := c.Query("audioTrack"); audioTrackStr != "" {
		if idx, err := parseInt(audioTrackStr); err == nil && idx >= 0 {
			audioTrackIndex = int(idx)
		}
	}

	// Parse preferred audio language (ISO 639-2 code, e.g., "eng", "spa", "fra")
	preferredAudioLanguage := c.Query("audioLanguage")

	// Parse client codec capabilities from query params first (passed from master playlist)
	// Fall back to headers if not present (direct requests)
	supportedVideoCodecs := parseCommaSeparatedHeader(c.Query("codecs"))
	if len(supportedVideoCodecs) == 0 {
		supportedVideoCodecs = parseCommaSeparatedHeader(c.GetHeader("X-Supported-Video-Codecs"))
	}
	supportedContainers := parseCommaSeparatedHeader(c.GetHeader("X-Supported-Containers"))

	// Parse strategy from query params (passed from master playlist for consistency)
	strategyHint := c.Query("strategy")

	// Use the serve manifest use case
	response, err := h.serveManifestUseCase.Execute(c.Request.Context(), transcode.ServeManifestRequest{
		MediaID:                 mediaID,
		Quality:                 quality,
		OutputDir:               h.outputDir,
		StartPosition:           startPosition,
		AudioTrackIndex:         audioTrackIndex,
		PreferredAudioLanguage:  preferredAudioLanguage,
		SupportedVideoCodecs:    supportedVideoCodecs,
		SupportedContainers:     supportedContainers,
		StrategyHint:            strategyHint,
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	// Handle response based on strategy
	switch response.Strategy {
	case transcode.StrategyServe:
		// Manifest generated - serve it with audio track parameter injected into segment URLs
		c.Header("Content-Type", "application/vnd.apple.mpegurl")

		// Include transcode session ID for frontend analytics correlation
		if response.SessionID != "" {
			c.Header("X-Session-ID", response.SessionID)
		}

		// Process the playlist before serving:
		// 1. Ensure #EXT-X-PLAYLIST-TYPE:EVENT is present (segment muxer omits it)
		// 2. Fix #EXT-X-MEDIA-SEQUENCE to 0 (segment muxer increments it, confusing HLS.js)
		// 3. Inject audio track parameter into segment URLs if multi-audio
		content, err := processPlaylistForServing(response.ManifestPath, audioTrackIndex)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to process playlist")
			return
		}
		c.Data(http.StatusOK, "application/vnd.apple.mpegurl", content)

		// Heartbeat: Update session last accessed time on playlist request.
		// HLS.js periodically refreshes the playlist during playback, which keeps
		// the session alive even when segment requests are briefly paused.
		if session, err := h.sessionManager.GetSession(mediaID, quality, audioTrackIndex); err == nil {
			session.UpdateLastAccessed()
		}
		h.queue.RecordAccess(mediaID, quality)

	case transcode.StrategyDirectPlay:
		// Video is compatible - redirect to direct stream
		// Frontend expects 302 redirect for direct play
		c.Redirect(http.StatusFound, response.DirectPlayURL)

	default:
		// Should never reach here with new segment-based system
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unknown streaming strategy")
	}
}

// ServeHLSSegment serves HLS segment files from progressive transcode sessions.
//
// @Summary Serve HLS segment
// @Description Serves HLS segment files (.ts) from progressive transcoding sessions
// @Tags transcode
// @Produce video/mp2t
// @Param id path int true "Media ID"
// @Param quality path string true "Quality level (360p, 720p, 1080p, 4k)"
// @Param filename path string true "Segment filename (e.g., seg_000123.ts)"
// @Success 200 {file} file "HLS segment file"
// @Failure 400 {object} handlers.APIError
// @Failure 404 {object} handlers.APIError
// @Failure 500 {object} handlers.APIError
// @Router /api/media/{id}/hls/{quality}/{filename} [get]
func (h *TranscodeHandler) ServeHLSSegment(c *gin.Context) {
	mediaIDStr := c.Param("id")
	quality := c.Param("quality")
	filename := c.Param("filename")

	// Parse media ID
	mediaID, err := parseID(mediaIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid media ID")
		return
	}

	// Parse optional audio track index for multi-audio selection
	// This must match the audioTrack used when creating the session via ServePlaylist
	audioTrackIndex := -1
	if audioTrackStr := c.Query("audioTrack"); audioTrackStr != "" {
		if idx, err := parseInt(audioTrackStr); err == nil && idx >= 0 {
			audioTrackIndex = int(idx)
		}
	}

	// Get active transcode session
	// audioTrackIndex is used to find the correct session for multi-audio support
	session, err := h.sessionManager.GetSession(mediaID, quality, audioTrackIndex)
	if err != nil {
		respondError(c, http.StatusNotFound, "NOT_FOUND", "No active transcode session")
		return
	}

	// Handle init segment for fMP4
	if filename == segment.InitFilename {
		initPath, err := session.WaitForInitSegment(10 * time.Second)
		if err != nil {
			respondError(c, http.StatusRequestTimeout, "INIT_SEGMENT_NOT_AVAILABLE", "Init segment not available")
			return
		}
		session.UpdateLastAccessed()
		h.queue.RecordAccess(mediaID, quality)
		c.Header("Content-Type", "video/mp4")
		c.File(initPath)
		return
	}

	// Parse segment number from filename
	segmentNum := segment.ParseNumber(filename)
	if segmentNum < 0 {
		respondError(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid segment filename")
		return
	}

	// Wait for segment to be generated (30 second timeout)
	segmentPath, err := session.WaitForSegment(segmentNum, 30*time.Second)
	if err != nil {
		respondError(c, http.StatusRequestTimeout, "SEGMENT_NOT_AVAILABLE___TRANSCODING_MAY_BE_SLOW_OR_FAILED", "Segment not available - transcoding may be slow or failed")
		return
	}

	// Update session last accessed time
	session.UpdateLastAccessed()
	h.queue.RecordAccess(mediaID, quality)

	// Serve the segment with appropriate content type
	// fMP4 segments use video/mp4, MPEG-TS uses video/mp2t
	contentType := "video/mp4"
	if strings.HasSuffix(segmentPath, ".ts") {
		contentType = "video/mp2t"
	}
	c.Header("Content-Type", contentType)
	c.File(segmentPath)
}

// ServeMasterPlaylist serves an HLS master playlist with the recommended quality variant.
// Uses client capabilities to determine optimal quality, returning a single-variant playlist.
//
// @Summary Serve HLS master playlist
// @Description Serves an HLS master playlist (.m3u8) with the optimal quality based on client capabilities.
// @Description Uses screen size, bandwidth, and codec support to recommend the best quality.
// @Description If the video is compatible for direct play (right codec, audio, container), returns 302 redirect.
// @Tags transcode
// @Produce application/vnd.apple.mpegurl,application/json
// @Param id path int true "Media ID"
// @Param start query number false "Start position in seconds for seeking"
// @Param screenWidth query int false "Client screen width in pixels"
// @Param screenHeight query int false "Client screen height in pixels"
// @Param bandwidth query int false "Estimated bandwidth in bits per second"
// @Param codecs query string false "Comma-separated list of supported codecs (h264,h265,vp9,av1)"
// @Param quality query string false "Override: force specific quality (e.g., 4k-25m, 1080p-10m)"
// @Success 200 {file} file "HLS master playlist with recommended quality"
// @Success 302 "Redirect to direct stream (for compatible files)"
// @Failure 400 {object} handlers.APIError
// @Failure 404 {object} handlers.APIError
// @Failure 500 {object} handlers.APIError
// @Router /api/media/{id}/hls/master.m3u8 [get]
func (h *TranscodeHandler) ServeMasterPlaylist(c *gin.Context) {
	mediaID, err := parseID(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid media ID")
		return
	}

	// Parse client codec capabilities from query params (preferred) or headers (fallback)
	// Query params are more reliable for cross-origin requests with redirect: 'manual'
	supportedVideoCodecs := parseCommaSeparatedHeader(c.Query("codecs"))
	if len(supportedVideoCodecs) == 0 {
		supportedVideoCodecs = parseCommaSeparatedHeader(c.GetHeader("X-Supported-Video-Codecs"))
	}
	supportedContainers := parseCommaSeparatedHeader(c.Query("containers"))
	if len(supportedContainers) == 0 {
		supportedContainers = parseCommaSeparatedHeader(c.GetHeader("X-Supported-Containers"))
	}

	// Parse optional audio track index for multi-audio selection
	// -1 means use default (first audio track), >= 0 is the FFmpeg stream index
	audioTrackIndex := -1
	if audioTrackStr := c.Query("audioTrack"); audioTrackStr != "" {
		if idx, err := parseInt(audioTrackStr); err == nil && idx >= 0 {
			audioTrackIndex = int(idx)
		}
	}

	// Parse client capabilities for quality recommendation
	var screenWidth, screenHeight int
	var bandwidth int64
	if w := c.Query("screenWidth"); w != "" {
		if parsed, err := parseInt(w); err == nil {
			screenWidth = parsed
		}
	}
	if h := c.Query("screenHeight"); h != "" {
		if parsed, err := parseInt(h); err == nil {
			screenHeight = parsed
		}
	}
	if b := c.Query("bandwidth"); b != "" {
		if parsed, err := parseInt(b); err == nil {
			bandwidth = int64(parsed)
		}
	}

	// Parse optional quality override (user manually selected a quality)
	qualityOverride := c.Query("quality")

	// Parse HDR display capability from client
	// If true, client has an HDR-capable display and can play HDR content natively
	clientSupportsHDR := c.Query("hdrDisplay") == "true"

	// Parse preferred audio language (ISO 639-2 code, e.g., "eng", "spa", "fra")
	preferredAudioLanguage := c.Query("audioLanguage")

	// Parse preferred subtitle language (ISO 639-2 code or "off")
	preferredSubtitleLanguage := c.Query("subtitleLanguage")

	// Use the serve master playlist use case
	response, err := h.serveMasterPlaylistUseCase.Execute(c.Request.Context(), transcode.ServeMasterPlaylistRequest{
		MediaID:                  mediaID,
		SupportedVideoCodecs:     supportedVideoCodecs,
		SupportedContainers:      supportedContainers,
		StartPosition:            c.Query("start"),
		AudioTrackIndex:          audioTrackIndex,
		PreferredAudioLanguage:   preferredAudioLanguage,
		PreferredSubtitleLanguage: preferredSubtitleLanguage,
		ScreenWidth:              screenWidth,
		ScreenHeight:             screenHeight,
		Bandwidth:                bandwidth,
		QualityOverride:          qualityOverride,
		ClientSupportsHDR:        clientSupportsHDR,
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	// Handle response based on strategy
	switch response.Strategy {
	case transcode.StrategyMasterDirectPlay:
		// Video is compatible - redirect to direct stream
		c.Redirect(http.StatusFound, response.DirectPlayURL)

	case transcode.StrategyServePlaylist:
		// Serve the generated master playlist
		c.Header("Content-Type", "application/vnd.apple.mpegurl")
		c.Header("Cache-Control", "no-cache")

		// Include available qualities as JSON header for frontend quality picker
		if len(response.AvailableQualities) > 0 {
			qualitiesJSON, err := json.Marshal(response.AvailableQualities)
			if err == nil {
				c.Header("X-Available-Qualities", string(qualitiesJSON))
			}
		}

		c.String(http.StatusOK, response.PlaylistContent)

	default:
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unknown streaming strategy")
	}
}

// ServeSubtitle serves a subtitle WebVTT file for HLS playback.
// Uses subtitle-extractor for fast extraction with FFmpeg fallback.
//
// @Summary Serve subtitle WebVTT file
// @Description Streams a text subtitle as WebVTT for HLS playback (fast demux, no full file scan)
// @Tags transcode
// @Produce text/vtt
// @Param id path int true "Media ID"
// @Param trackIndex path int true "Subtitle track index (0-based, among text subtitles only)"
// @Param start query number false "Start position in seconds (for seeking)"
// @Success 200 {file} file "WebVTT subtitle file"
// @Failure 400 {object} handlers.APIError
// @Failure 404 {object} handlers.APIError
// @Failure 500 {object} handlers.APIError
// @Router /api/media/{id}/hls/subtitle/{trackIndex}/subtitles.vtt [get]
func (h *TranscodeHandler) ServeSubtitle(c *gin.Context) {
	mediaID, err := parseID(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid media ID")
		return
	}

	relativeIndex, err := parseInt(c.Param("trackIndex"))
	if err != nil || relativeIndex < 0 {
		respondError(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid track index")
		return
	}

	// Find text track at this relative index
	targetTrack, err := h.getTracksUseCase.GetSubtitleTrackByRelativeIndex(c.Request.Context(), mediaID, relativeIndex, false)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get subtitle tracks")
		return
	}
	if targetTrack == nil {
		respondError(c, http.StatusNotFound, "NOT_FOUND", "Subtitle track not found")
		return
	}

	// Get media file path
	mediaResp, err := h.getMediaUseCase.Execute(c.Request.Context(), mediaID)
	if err != nil {
		respondError(c, http.StatusNotFound, "NOT_FOUND", "Media not found")
		return
	}

	// Use the converter which handles subtitle-extractor with FFmpeg fallback
	vttPath, err := h.subtitleConverter.ExtractAndConvert(c.Request.Context(), mediaID, mediaResp.FilePath, *targetTrack.StreamIndex)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to extract subtitle")
		return
	}

	// Read and serve the WebVTT content
	vttContent, err := subtitles.GetWebVTTContent(vttPath)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to read subtitle file")
		return
	}

	// Serve the WebVTT content
	c.Header("Content-Type", "text/vtt; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=86400") // Cache for 24 hours
	c.String(http.StatusOK, vttContent)
}

// ServeHeartbeat handles keepalive requests from the frontend.
// Updates the session's last accessed time to prevent idle timeout
// during pauses or brief network interruptions.
//
// @Summary Keep transcode session alive
// @Description Lightweight endpoint to prevent idle session timeout during pauses
// @Tags transcode
// @Param id path int true "Media ID"
// @Param quality query string true "Quality level (e.g., 1080p-10m)"
// @Param audioTrack query int false "Audio track index"
// @Success 200 "Session alive"
// @Success 404 "Session not found"
// @Failure 400 {object} handlers.APIError
// @Router /api/media/{id}/hls/heartbeat [get]
func (h *TranscodeHandler) ServeHeartbeat(c *gin.Context) {
	mediaID, err := parseID(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid media ID")
		return
	}

	quality := c.Query("quality")
	if quality == "" {
		c.Status(http.StatusOK)
		return
	}

	audioTrackIndex := -1
	if audioTrackStr := c.Query("audioTrack"); audioTrackStr != "" {
		if idx, err := parseInt(audioTrackStr); err == nil && idx >= 0 {
			audioTrackIndex = int(idx)
		}
	}

	session, err := h.sessionManager.GetSession(mediaID, quality, audioTrackIndex)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	session.UpdateLastAccessed()
	h.queue.RecordAccess(mediaID, quality)

	c.Status(http.StatusOK)
}

// processPlaylistForServing reads an HLS playlist and sanitises it for HLS.js:
//  1. Injects #EXT-X-PLAYLIST-TYPE:EVENT if missing (segment muxer with +live omits it)
//  2. Forces #EXT-X-MEDIA-SEQUENCE:0 (segment muxer increments it, causing live-sync confusion)
//  3. Appends ?audioTrack=X to segment URLs when multi-audio is in use
func processPlaylistForServing(playlistPath string, audioTrackIndex int) ([]byte, error) {
	file, err := os.Open(playlistPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var result strings.Builder
	scanner := bufio.NewScanner(file)

	hasPlaylistType := false
	hasMediaSequence := false
	needsNewline := false

	for scanner.Scan() {
		line := scanner.Text()

		// Track whether the playlist already has EXT-X-PLAYLIST-TYPE
		if strings.HasPrefix(line, "#EXT-X-PLAYLIST-TYPE:") {
			hasPlaylistType = true
		}

		// Override media sequence to always start at 0
		if strings.HasPrefix(line, "#EXT-X-MEDIA-SEQUENCE:") {
			line = "#EXT-X-MEDIA-SEQUENCE:0"
			hasMediaSequence = true
		}

		// Append audio track parameter to segment URLs
		if audioTrackIndex > 0 &&
			!strings.HasPrefix(line, "#") &&
			(strings.HasSuffix(line, ".ts") || strings.HasSuffix(line, ".m4s")) {
			line = fmt.Sprintf("%s?audioTrack=%d", line, audioTrackIndex)
		}

		if needsNewline {
			result.WriteString("\n")
		}
		result.WriteString(line)
		needsNewline = true
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Inject #EXT-X-PLAYLIST-TYPE:EVENT right after #EXTM3U if missing
	if !hasPlaylistType {
		content := result.String()
		content = strings.Replace(content, "#EXTM3U", "#EXTM3U\n#EXT-X-PLAYLIST-TYPE:EVENT", 1)
		return []byte(content), nil
	}

	// Ensure media sequence line exists (should always be present)
	if !hasMediaSequence {
		content := result.String()
		content = strings.Replace(content, "#EXTM3U", "#EXTM3U\n#EXT-X-MEDIA-SEQUENCE:0", 1)
		return []byte(content), nil
	}

	return []byte(result.String()), nil
}
