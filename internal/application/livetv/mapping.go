package livetv

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mantonx/viewra/internal/domain/library"
	"github.com/mantonx/viewra/internal/domain/livetv"
	"github.com/mantonx/viewra/internal/infrastructure/database/unified"
	infraLivetv "github.com/mantonx/viewra/internal/infrastructure/livetv"
)

type EPGChannelInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ChannelMappingResponse struct {
	XmltvChannelID string `json:"xmltv_channel_id"`
	ChannelID      int64  `json:"channel_id,omitempty"`
	ChannelName    string `json:"channel_name,omitempty"`
	EPGChannelName string `json:"epg_channel_name,omitempty"`
}

type ListMappingsResponse struct {
	EPGChannels []EPGChannelInfo       `json:"epg_channels"`
	Mappings    []ChannelMappingResponse `json:"mappings"`
	Channels    []ChannelResponse      `json:"channels"`
	LibraryPath string                  `json:"library_path"`
}

type SetMappingRequest struct {
	XmltvChannelID string `json:"xmltv_channel_id"`
	ChannelID      int64  `json:"channel_id"`
}

type ListMappingsUseCase struct {
	channelRepo livetv.ChannelRepository
	libraryRepo library.Repository
	querier     unified.Querier
	httpClient  *http.Client
}

type SetMappingUseCase struct {
	libraryRepo library.Repository
	querier     unified.Querier
}

type DeleteMappingUseCase struct {
	querier unified.Querier
}

func NewListMappingsUseCase(channelRepo livetv.ChannelRepository, libraryRepo library.Repository, querier unified.Querier) *ListMappingsUseCase {
	return &ListMappingsUseCase{
		channelRepo: channelRepo,
		libraryRepo: libraryRepo,
		querier:     querier,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

func NewSetMappingUseCase(libraryRepo library.Repository, querier unified.Querier) *SetMappingUseCase {
	return &SetMappingUseCase{
		libraryRepo: libraryRepo,
		querier:     querier,
	}
}

func NewDeleteMappingUseCase(querier unified.Querier) *DeleteMappingUseCase {
	return &DeleteMappingUseCase{
		querier: querier,
	}
}

func (uc *ListMappingsUseCase) Execute(ctx context.Context, libraryID int64) (ListMappingsResponse, error) {
	lib, err := uc.libraryRepo.GetByID(ctx, libraryID)
	if err != nil {
		return ListMappingsResponse{}, fmt.Errorf("failed to get library: %w", err)
	}

	// Get available M3U channels
	channels, err := uc.channelRepo.ListByLibrary(ctx, libraryID)
	if err != nil {
		return ListMappingsResponse{}, fmt.Errorf("failed to list channels: %w", err)
	}

	// Get existing mappings with channel info
	mappingRows, err := uc.querier.ListChannelEPGMappings(ctx, libraryID)
	if err != nil {
		return ListMappingsResponse{}, fmt.Errorf("failed to list mappings: %w", err)
	}

	mappings := make([]ChannelMappingResponse, len(mappingRows))
	for i, m := range mappingRows {
		mappings[i] = ChannelMappingResponse{
			XmltvChannelID: m.XmltvChannelID,
			ChannelID:      m.ChannelID,
			ChannelName:    m.ChannelName,
		}
	}

	// Build map of existing mappings for quick lookup
	mappedXMLTV := make(map[string]bool)
	for _, m := range mappings {
		mappedXMLTV[m.XmltvChannelID] = true
	}

	// Extract EPG channels from XMLTV source
	var epgChannels []EPGChannelInfo
	xmltvPath := ""
	if lib.MonitoringConfig != nil && lib.MonitoringConfig.XmltvURL != "" {
		xmltvPath = lib.MonitoringConfig.XmltvURL
	} else {
		// Auto-detect guide.xml in library directory
		xmltvPath = findXMLTVFile(lib.Path)
	}
	if xmltvPath != "" {
		epgChannels = uc.extractEPGChannels(ctx, xmltvPath)
	}

	// Annotate mappings with EPG channel names
	epgNameMap := make(map[string]string)
	for _, ec := range epgChannels {
		epgNameMap[ec.ID] = ec.Name
	}
	for i, m := range mappings {
		if name, ok := epgNameMap[m.XmltvChannelID]; ok {
			mappings[i].EPGChannelName = name
		}
	}

	// Include unmapped EPG channels as unmapped entries in the mappings list
	for _, ec := range epgChannels {
		if !mappedXMLTV[ec.ID] {
			mappings = append(mappings, ChannelMappingResponse{
				XmltvChannelID: ec.ID,
				EPGChannelName: ec.Name,
			})
		}
	}

	channelResponses := make([]ChannelResponse, len(channels))
	for i, ch := range channels {
		channelResponses[i] = toChannelResponse(ch)
	}

	return ListMappingsResponse{
		EPGChannels: epgChannels,
		Mappings:    mappings,
		Channels:    channelResponses,
		LibraryPath: lib.Path,
	}, nil
}

func (uc *ListMappingsUseCase) extractEPGChannels(ctx context.Context, xmltvPath string) []EPGChannelInfo {
	reader, err := openXMLTVSource(xmltvPath, uc.httpClient)
	if err != nil {
		return nil
	}

	result, err := infraLivetv.ParseXMLTV(reader)
	if err != nil {
		return nil
	}

	channels := make([]EPGChannelInfo, 0, len(result.ChannelMap))
	for id, name := range result.ChannelMap {
		if name == "" {
			name = id
		}
		channels = append(channels, EPGChannelInfo{ID: id, Name: name})
	}
	return channels
}

func (uc *SetMappingUseCase) Execute(ctx context.Context, libraryID int64, req SetMappingRequest) (ChannelMappingResponse, error) {
	// Check if mapping already exists
	_, err := uc.querier.GetChannelEPGMapping(ctx, unified.GetChannelEPGMappingParams{
		LibraryID:      libraryID,
		XmltvChannelID: req.XmltvChannelID,
	})
	if err == nil {
		// Update existing mapping
		result, err := uc.querier.UpdateChannelEPGMapping(ctx, unified.UpdateChannelEPGMappingParams{
			ChannelID:      req.ChannelID,
			LibraryID:      libraryID,
			XmltvChannelID: req.XmltvChannelID,
		})
		if err != nil {
			return ChannelMappingResponse{}, fmt.Errorf("failed to update mapping: %w", err)
		}
		return ChannelMappingResponse{
			XmltvChannelID: result.XmltvChannelID,
			ChannelID:      result.ChannelID,
		}, nil
	}

	// Create new mapping
	result, err := uc.querier.CreateChannelEPGMapping(ctx, unified.CreateChannelEPGMappingParams{
		LibraryID:      libraryID,
		XmltvChannelID: req.XmltvChannelID,
		ChannelID:      req.ChannelID,
	})
	if err != nil {
		return ChannelMappingResponse{}, fmt.Errorf("failed to create mapping: %w", err)
	}

	return ChannelMappingResponse{
		XmltvChannelID: result.XmltvChannelID,
		ChannelID:      result.ChannelID,
	}, nil
}

func (uc *DeleteMappingUseCase) Execute(ctx context.Context, libraryID int64, xmltvChannelID string) error {
	return uc.querier.DeleteChannelEPGMapping(ctx, unified.DeleteChannelEPGMappingParams{
		LibraryID:      libraryID,
		XmltvChannelID: xmltvChannelID,
	})
}

func openXMLTVSource(path string, httpClient *http.Client) (io.ReadCloser, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Try as URL
			req, err := http.NewRequest(http.MethodGet, path, nil)
			if err != nil {
				return nil, err
			}
			resp, err := httpClient.Do(req)
			if err != nil {
				return nil, err
			}
			return resp.Body, nil
		}
		return nil, err
	}

	if info.IsDir() {
		entries, err := filepath.Glob(filepath.Join(path, "guide.xml"))
		if err != nil {
			return nil, err
		}
		if len(entries) == 0 {
			entries, err = filepath.Glob(filepath.Join(path, "*.xml"))
			if err != nil {
				return nil, err
			}
		}
		if len(entries) == 0 {
			return nil, fmt.Errorf("no XMLTV files found in %s", path)
		}
		return os.Open(entries[0])
	}

	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".xml" || ext == ".xmltv" {
		return os.Open(path)
	}

	// Unknown path type, try as URL
	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func findXMLTVFile(path string) string {
	entries, err := filepath.Glob(filepath.Join(path, "guide.xml"))
	if err != nil || len(entries) == 0 {
		return ""
	}
	return entries[0]
}
