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

type ScanEPGUseCase struct {
	channelRepo livetv.ChannelRepository
	programRepo livetv.ProgramRepository
	libraryRepo library.Repository
	querier     unified.Querier
	httpClient  *http.Client
}

func NewScanEPGUseCase(channelRepo livetv.ChannelRepository, programRepo livetv.ProgramRepository, libraryRepo library.Repository, querier unified.Querier) *ScanEPGUseCase {
	return &ScanEPGUseCase{
		channelRepo: channelRepo,
		programRepo: programRepo,
		libraryRepo: libraryRepo,
		querier:     querier,
		httpClient:  &http.Client{Timeout: 60 * time.Second},
	}
}

func (uc *ScanEPGUseCase) Execute(ctx context.Context, libraryID int64) (ScanResponse, error) {
	lib, err := uc.libraryRepo.GetByID(ctx, libraryID)
	if err != nil {
		return ScanResponse{}, fmt.Errorf("failed to get library: %w", err)
	}

	xmltvURL := ""
	if lib.MonitoringConfig != nil {
		xmltvURL = lib.MonitoringConfig.XmltvURL
	}
	if xmltvURL == "" {
		xmltvURL = findXMLTVFile(lib.Path)
		if xmltvURL != "" {
			if lib.MonitoringConfig == nil {
				lib.MonitoringConfig = &library.MonitoringConfig{}
			}
			lib.MonitoringConfig.XmltvURL = xmltvURL
			if err := uc.libraryRepo.UpdateMonitoring(ctx, lib.ID, lib.MonitoringEnabled, lib.MonitoringConfig); err != nil {
				// Log warning but continue with the detected path
			}
		}
	}
	if xmltvURL == "" {
		return ScanResponse{}, fmt.Errorf("library has no XMLTV path/URL configured and no guide.xml found in %s", lib.Path)
	}

	var reader io.Reader
	var sourceDescription string

	info, err := os.Stat(xmltvURL)
	if err != nil {
		if os.IsNotExist(err) {
			return uc.fetchAndParseURL(ctx, xmltvURL, libraryID)
		}
		return ScanResponse{}, fmt.Errorf("failed to stat XMLTV path: %w", err)
	}

	if info.IsDir() {
		sourceDescription = fmt.Sprintf("directory %s", xmltvURL)
		entries, err := filepath.Glob(filepath.Join(xmltvURL, "guide.xml"))
		if err != nil {
			return ScanResponse{}, fmt.Errorf("failed to glob guide.xml: %w", err)
		}
		if len(entries) == 0 {
			entries, err = filepath.Glob(filepath.Join(xmltvURL, "*.xml"))
			if err != nil {
				return ScanResponse{}, fmt.Errorf("failed to glob xml files: %w", err)
			}
		}
		if len(entries) == 0 {
			return ScanResponse{}, fmt.Errorf("no XMLTV files found in %s", xmltvURL)
		}
		f, err := os.Open(entries[0])
		if err != nil {
			return ScanResponse{}, fmt.Errorf("failed to open %s: %w", entries[0], err)
		}
		reader = f
		defer f.Close()
	} else {
		ext := strings.ToLower(filepath.Ext(xmltvURL))
		if ext == ".xml" || ext == ".xmltv" {
			sourceDescription = fmt.Sprintf("file %s", xmltvURL)
			f, err := os.Open(xmltvURL)
			if err != nil {
				return ScanResponse{}, fmt.Errorf("failed to open %s: %w", xmltvURL, err)
			}
			reader = f
			defer f.Close()
		} else {
			return uc.fetchAndParseURL(ctx, xmltvURL, libraryID)
		}
	}

	return uc.parseAndImport(ctx, reader, libraryID, sourceDescription)
}

func (uc *ScanEPGUseCase) fetchAndParseURL(ctx context.Context, url string, libraryID int64) (ScanResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ScanResponse{}, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := uc.httpClient.Do(req)
	if err != nil {
		return ScanResponse{}, fmt.Errorf("failed to fetch XMLTV: %w", err)
	}
	defer resp.Body.Close()

	return uc.parseAndImport(ctx, resp.Body, libraryID, "URL")
}

func (uc *ScanEPGUseCase) parseAndImport(ctx context.Context, reader io.Reader, libraryID int64, sourceDescription string) (ScanResponse, error) {
	xmltvResult, err := infraLivetv.ParseXMLTV(reader)
	if err != nil {
		return ScanResponse{}, fmt.Errorf("failed to parse XMLTV: %w", err)
	}

	channels, err := uc.channelRepo.ListByLibrary(ctx, libraryID)
	if err != nil {
		return ScanResponse{}, fmt.Errorf("failed to list channels: %w", err)
	}

	// Build mapping: XMLTV channel ID -> database channel ID
	// First use explicit mapping table, then fall back to channel.EPGChannelID
	epgToChannelID := make(map[string]int64)

	// 1. Load explicit mappings from database
	mappings, err := uc.querier.GetEPGMappingForLibrary(ctx, libraryID)
	if err == nil {
		for _, m := range mappings {
			epgToChannelID[m.XmltvChannelID] = m.ChannelID
		}
	}

	// 2. Fall back to channel's EPGChannelID for channels without explicit mapping
	for _, ch := range channels {
		if ch.EPGChannelID != "" {
			if _, exists := epgToChannelID[ch.EPGChannelID]; !exists {
				epgToChannelID[ch.EPGChannelID] = ch.ID
			}
		}
	}

	programs := infraLivetv.ToDomainPrograms(xmltvResult.Programs, epgToChannelID)

	if err := uc.programRepo.DeleteByLibrary(ctx, libraryID); err != nil {
		return ScanResponse{}, fmt.Errorf("failed to clear old EPG data: %w", err)
	}

	if err := uc.programRepo.BulkCreate(ctx, programs); err != nil {
		return ScanResponse{}, fmt.Errorf("failed to bulk create programs: %w", err)
	}

	return ScanResponse{
		Message: fmt.Sprintf("Imported %d EPG programs for %d channels from %s", len(programs), len(epgToChannelID), sourceDescription),
	}, nil
}
