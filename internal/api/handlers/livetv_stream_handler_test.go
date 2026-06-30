package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mantonx/viewra/internal/infrastructure/transcoding/session"
)

func TestLiveTvStreamHandler_GetPlaylist(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		libraryID      string
		channelID      string
		getChannelURL  string
		expectedStatus int
	}{
		{
			name:           "invalid library ID",
			libraryID:      "abc",
			channelID:      "1",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid channel ID",
			libraryID:      "1",
			channelID:      "abc",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "no session and no getChannelByID",
			libraryID:      "1",
			channelID:      "1",
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &session.Manager{}
			handler := NewLiveTvStreamHandler(mgr, nil)
			c, w := setupTestContext(http.MethodGet, "/api/libraries/"+tt.libraryID+"/livetv/stream/"+tt.channelID+"/playlist.m3u8", nil)
			c.Params = []gin.Param{
				{Key: "libraryId", Value: tt.libraryID},
				{Key: "channelId", Value: tt.channelID},
			}

			handler.GetPlaylist(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestLiveTvStreamHandler_GetSegment(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		libraryID      string
		channelID      string
		filename       string
		expectedStatus int
	}{
		{
			name:           "invalid library ID",
			libraryID:      "abc",
			channelID:      "1",
			filename:       "seg_000001.ts",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid channel ID",
			libraryID:      "1",
			channelID:      "abc",
			filename:       "seg_000001.ts",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing filename",
			libraryID:      "1",
			channelID:      "1",
			filename:       "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid extension",
			libraryID:      "1",
			channelID:      "1",
			filename:       "playlist.m3u8",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "no active session",
			libraryID:      "1",
			channelID:      "1",
			filename:       "seg_000001.ts",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &session.Manager{}
			handler := NewLiveTvStreamHandler(mgr, nil)
			c, w := setupTestContext(http.MethodGet, "/api/libraries/"+tt.libraryID+"/livetv/stream/"+tt.channelID+"/"+tt.filename, nil)
			c.Params = []gin.Param{
				{Key: "libraryId", Value: tt.libraryID},
				{Key: "channelId", Value: tt.channelID},
				{Key: "filename", Value: tt.filename},
			}

			handler.GetSegment(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestLiveTvStreamHandler_Pause(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		libraryID      string
		channelID      string
		expectedStatus int
	}{
		{
			name:           "invalid library ID",
			libraryID:      "abc",
			channelID:      "1",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid channel ID",
			libraryID:      "1",
			channelID:      "abc",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "no active session",
			libraryID:      "1",
			channelID:      "1",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &session.Manager{}
			handler := NewLiveTvStreamHandler(mgr, nil)
			c, w := setupTestContext(http.MethodPost, "/api/libraries/"+tt.libraryID+"/livetv/stream/"+tt.channelID+"/pause", nil)
			c.Params = []gin.Param{
				{Key: "libraryId", Value: tt.libraryID},
				{Key: "channelId", Value: tt.channelID},
			}

			handler.Pause(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestLiveTvStreamHandler_Resume(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		libraryID      string
		channelID      string
		expectedStatus int
	}{
		{
			name:           "invalid library ID",
			libraryID:      "abc",
			channelID:      "1",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid channel ID",
			libraryID:      "1",
			channelID:      "abc",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "no active session",
			libraryID:      "1",
			channelID:      "1",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &session.Manager{}
			handler := NewLiveTvStreamHandler(mgr, nil)
			c, w := setupTestContext(http.MethodPost, "/api/libraries/"+tt.libraryID+"/livetv/stream/"+tt.channelID+"/resume", nil)
			c.Params = []gin.Param{
				{Key: "libraryId", Value: tt.libraryID},
				{Key: "channelId", Value: tt.channelID},
			}

			handler.Resume(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestLiveTvStreamHandler_Stop(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		libraryID      string
		channelID      string
		expectedStatus int
	}{
		{
			name:           "invalid library ID",
			libraryID:      "abc",
			channelID:      "1",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid channel ID",
			libraryID:      "1",
			channelID:      "abc",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "no active session",
			libraryID:      "1",
			channelID:      "1",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &session.Manager{}
			handler := NewLiveTvStreamHandler(mgr, nil)
			c, w := setupTestContext(http.MethodPost, "/api/libraries/"+tt.libraryID+"/livetv/stream/"+tt.channelID+"/stop", nil)
			c.Params = []gin.Param{
				{Key: "libraryId", Value: tt.libraryID},
				{Key: "channelId", Value: tt.channelID},
			}

			handler.Stop(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestLiveTvStreamHandler_StreamStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		libraryID      string
		channelID      string
		expectedStatus int
		expectActive   bool
	}{
		{
			name:           "invalid library ID",
			libraryID:      "abc",
			channelID:      "1",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid channel ID",
			libraryID:      "1",
			channelID:      "abc",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "no active session",
			libraryID:      "1",
			channelID:      "1",
			expectedStatus: http.StatusOK,
			expectActive:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &session.Manager{}
			handler := NewLiveTvStreamHandler(mgr, nil)
			c, w := setupTestContext(http.MethodGet, "/api/libraries/"+tt.libraryID+"/livetv/stream/"+tt.channelID+"/status", nil)
			c.Params = []gin.Param{
				{Key: "libraryId", Value: tt.libraryID},
				{Key: "channelId", Value: tt.channelID},
			}

			handler.StreamStatus(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var resp StreamStatusResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to parse response: %v", err)
				}
				if resp.Active != tt.expectActive {
					t.Errorf("Active = %v, want %v", resp.Active, tt.expectActive)
				}
				if !resp.Active && resp.State != "inactive" {
					t.Errorf("State = %q, want %q", resp.State, "inactive")
				}
			}
		})
	}
}


