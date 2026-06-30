package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	appLivetv "github.com/mantonx/viewra/internal/application/livetv"
	"github.com/mantonx/viewra/internal/domain/library"
	"github.com/mantonx/viewra/internal/domain/livetv"
)

type handlerMockChannelRepo struct {
	livetv.ChannelRepository
	channels []*livetv.Channel
}

func (m *handlerMockChannelRepo) ListByLibrary(ctx context.Context, libraryID int64) ([]*livetv.Channel, error) {
	return m.channels, nil
}

type handlerMockProgramRepo struct {
	livetv.ProgramRepository
	programs []*livetv.Program
}

func (m *handlerMockProgramRepo) GetCurrentByChannel(ctx context.Context, channelID int64, now time.Time) (*livetv.Program, error) {
	return nil, errors.New("no program")
}

func (m *handlerMockProgramRepo) ListByLibrary(ctx context.Context, libraryID int64, from, to time.Time) ([]*livetv.Program, error) {
	return m.programs, nil
}

func (m *handlerMockProgramRepo) GetByID(ctx context.Context, id int64) (*livetv.Program, error) {
	for _, p := range m.programs {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, nil
}

type handlerMockLibraryRepo struct {
	library.Repository
	lib *library.Library
}

func (m *handlerMockLibraryRepo) GetByID(ctx context.Context, id int64) (*library.Library, error) {
	return m.lib, nil
}

func TestLiveTvHandler_ListChannels(t *testing.T) {
	gin.SetMode(gin.TestMode)

	channelRepo := &handlerMockChannelRepo{channels: []*livetv.Channel{
		{ID: 1, LibraryID: 1, Name: "BBC One", StreamURL: "rtsp://example.com/bbc", Enabled: true},
	}}
	programRepo := &handlerMockProgramRepo{}
	listChannels := appLivetv.NewListChannelsUseCase(channelRepo, programRepo)

	handler := NewLiveTvHandler(listChannels, nil, nil, nil, nil, nil, nil)
	c, w := setupTestContext(http.MethodGet, "/api/libraries/1/livetv/channels", nil)
	c.Params = []gin.Param{{Key: "libraryId", Value: "1"}}

	handler.ListChannels(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	var resp appLivetv.ListChannelsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp.Channels) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(resp.Channels))
	}
	if resp.Channels[0].Name != "BBC One" {
		t.Errorf("Name = %q", resp.Channels[0].Name)
	}
}

func TestLiveTvHandler_ListChannels_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewLiveTvHandler(nil, nil, nil, nil, nil, nil, nil)
	c, w := setupTestContext(http.MethodGet, "/api/libraries/abc/livetv/channels", nil)
	c.Params = []gin.Param{{Key: "libraryId", Value: "abc"}}

	handler.ListChannels(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestLiveTvHandler_ListEPG(t *testing.T) {
	gin.SetMode(gin.TestMode)

	start := time.Date(2026, 6, 13, 4, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 13, 5, 0, 0, 0, time.UTC)
	programRepo := &handlerMockProgramRepo{programs: []*livetv.Program{
		{ID: 1, ChannelID: 1, Title: "Morning News", StartTime: start, EndTime: end},
	}}
	listEPG := appLivetv.NewListEPGUseCase(programRepo)

	handler := NewLiveTvHandler(nil, listEPG, nil, nil, nil, nil, nil)
	c, w := setupTestContext(http.MethodGet, "/api/libraries/1/livetv/epg", nil)
	c.Params = []gin.Param{{Key: "libraryId", Value: "1"}}

	handler.ListEPG(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	var resp appLivetv.ListEPGResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp.Programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(resp.Programs))
	}
}

func TestLiveTvHandler_ListEPG_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewLiveTvHandler(nil, nil, nil, nil, nil, nil, nil)
	c, w := setupTestContext(http.MethodGet, "/api/libraries/abc/livetv/epg", nil)
	c.Params = []gin.Param{{Key: "libraryId", Value: "abc"}}

	handler.ListEPG(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestLiveTvHandler_ListEPG_InvalidFromTime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewLiveTvHandler(nil, appLivetv.NewListEPGUseCase(&handlerMockProgramRepo{}), nil, nil, nil, nil, nil)
	c, w := setupTestContext(http.MethodGet, "/api/libraries/1/livetv/epg?from=invalid", nil)
	c.Params = []gin.Param{{Key: "libraryId", Value: "1"}}

	handler.ListEPG(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestLiveTvHandler_GetEPGProgram(t *testing.T) {
	gin.SetMode(gin.TestMode)

	start := time.Date(2026, 6, 13, 4, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 13, 5, 0, 0, 0, time.UTC)
	programRepo := &handlerMockProgramRepo{programs: []*livetv.Program{
		{ID: 1, ChannelID: 1, Title: "Morning News", StartTime: start, EndTime: end},
	}}
	listEPG := appLivetv.NewListEPGUseCase(programRepo)

	handler := NewLiveTvHandler(nil, listEPG, nil, nil, nil, nil, nil)
	c, w := setupTestContext(http.MethodGet, "/api/livetv/epg/1", nil)
	c.Params = []gin.Param{{Key: "programId", Value: "1"}}

	handler.GetEPGProgram(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	var resp appLivetv.ProgramResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Title != "Morning News" {
		t.Errorf("Title = %q", resp.Title)
	}
}

func TestLiveTvHandler_GetEPGProgram_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewLiveTvHandler(nil, nil, nil, nil, nil, nil, nil)
	c, w := setupTestContext(http.MethodGet, "/api/livetv/epg/abc", nil)
	c.Params = []gin.Param{{Key: "programId", Value: "abc"}}

	handler.GetEPGProgram(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestLiveTvHandler_ScanChannels_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewLiveTvHandler(nil, nil, nil, nil, nil, nil, nil)
	c, w := setupTestContext(http.MethodPost, "/api/libraries/abc/livetv/scan-channels", nil)
	c.Params = []gin.Param{{Key: "libraryId", Value: "abc"}}

	handler.ScanChannels(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestLiveTvHandler_ScanChannels_NoPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	scanChannels := appLivetv.NewScanChannelsUseCase(
		&handlerMockChannelRepo{},
		&handlerMockLibraryRepo{lib: &library.Library{ID: 1, Path: ""}},
	)
	handler := NewLiveTvHandler(nil, nil, scanChannels, nil, nil, nil, nil)
	c, w := setupTestContext(http.MethodPost, "/api/libraries/1/livetv/scan-channels", nil)
	c.Params = []gin.Param{{Key: "libraryId", Value: "1"}}

	handler.ScanChannels(c)

	if w.Code == http.StatusOK {
		t.Errorf("expected non-OK status, got %d", w.Code)
	}
}

func TestLiveTvHandler_ListMappings_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewLiveTvHandler(nil, nil, nil, nil, nil, nil, nil)
	c, w := setupTestContext(http.MethodGet, "/api/libraries/abc/livetv/mappings", nil)
	c.Params = []gin.Param{{Key: "libraryId", Value: "abc"}}

	handler.ListMappings(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestLiveTvHandler_SetMapping_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewLiveTvHandler(nil, nil, nil, nil, nil, nil, nil)
	c, w := setupTestContext(http.MethodPost, "/api/libraries/abc/livetv/mappings", nil)
	c.Params = []gin.Param{{Key: "libraryId", Value: "abc"}}

	handler.SetMapping(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestLiveTvHandler_DeleteMapping_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewLiveTvHandler(nil, nil, nil, nil, nil, nil, nil)
	c, w := setupTestContext(http.MethodDelete, "/api/libraries/abc/livetv/mappings/bbc1", nil)
	c.Params = []gin.Param{
		{Key: "libraryId", Value: "abc"},
		{Key: "xmltvChannelId", Value: "bbc1"},
	}

	handler.DeleteMapping(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestLiveTvHandler_DeleteMapping_MissingChannelID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewLiveTvHandler(nil, nil, nil, nil, nil, nil, nil)
	c, w := setupTestContext(http.MethodDelete, "/api/libraries/1/livetv/mappings/", nil)
	c.Params = []gin.Param{
		{Key: "libraryId", Value: "1"},
		{Key: "xmltvChannelId", Value: ""},
	}

	handler.DeleteMapping(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestLiveTvHandler_ScanEPG_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewLiveTvHandler(nil, nil, nil, nil, nil, nil, nil)
	c, w := setupTestContext(http.MethodPost, "/api/libraries/abc/livetv/scan-epg", nil)
	c.Params = []gin.Param{{Key: "libraryId", Value: "abc"}}

	handler.ScanEPG(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
