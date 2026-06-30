package execution

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mantonx/viewra/internal/application/library/scan/status"
	"github.com/mantonx/viewra/internal/domain/library"
	"github.com/mantonx/viewra/internal/domain/livetv"
	"github.com/mantonx/viewra/internal/domain/scanner"
	"github.com/mantonx/viewra/internal/infrastructure/database/unified"
	infraLivetv "github.com/mantonx/viewra/internal/infrastructure/livetv"
)

type LiveTVDeps struct {
	ChannelRepo livetv.ChannelRepository
	ProgramRepo livetv.ProgramRepository
	EPGQuerier  *unified.Querier
}

func RunLiveTVScan(ctx context.Context, deps *Deps, liveTVDeps *LiveTVDeps, jobID int64, lib *library.Library) {
	logger := deps.Logger

	// 1. Find M3U files in the library path
	m3uFiles := findM3UFiles(lib.Path)
	if len(m3uFiles) == 0 {
		status.CompleteWithError(ctx, deps.StatusDeps(), jobID, fmt.Errorf("no M3U files found in %s", lib.Path))
		return
	}

	filesFound := int64(len(m3uFiles))
	logger.Info("live TV discovery", "library_id", lib.ID, "m3u_files", filesFound)

	// Update progress: discovery phase with files found
	if err := deps.ProgressUpdater.NewProgressUpdate(jobID).
		Phase(scanner.ScanPhaseDiscovering).
		FilesFound(filesFound).
		DiscoveryDone().
		Update(ctx); err != nil {
		logger.Warn("failed to update discovery progress", "job_id", jobID, "error", err)
	}

	// 2. Parse each M3U file and import channels
	totalChannels := 0
	for _, m3uPath := range m3uFiles {
		f, err := os.Open(m3uPath)
		if err != nil {
			logger.Warn("failed to open M3U file, skipping", "path", m3uPath, "error", err)
			continue
		}

		entries, parseErr := infraLivetv.ParseM3U(f)
		f.Close()
		if parseErr != nil {
			logger.Warn("failed to parse M3U file, skipping", "path", m3uPath, "error", parseErr)
			continue
		}

		entries, pidErr := infraLivetv.CorrectSATIPPIDs(ctx, entries, lib, deps.MediaRepos.Library, liveTVDeps.ChannelRepo, &http.Client{Timeout: 30 * time.Second})
		if pidErr != nil {
			logger.Warn("failed to correct SAT>IP PIDs, continuing with original entries", "path", m3uPath, "error", pidErr)
		}

		for _, entry := range entries {
			ch := entry.ToDomainChannel(lib.ID)
			if upsertErr := liveTVDeps.ChannelRepo.Upsert(ctx, ch); upsertErr != nil {
				logger.Warn("failed to upsert channel", "name", ch.Name, "error", upsertErr)
				continue
			}
			totalChannels++
		}
	}

	if totalChannels == 0 {
		status.CompleteWithError(ctx, deps.StatusDeps(), jobID, fmt.Errorf("no channels imported from %d M3U files", filesFound))
		return
	}

	filesProcessed := filesFound

	// Update progress: processing phase
	if err := deps.ProgressUpdater.NewProgressUpdate(jobID).
		Phase(scanner.ScanPhaseProcessing).
		FilesFound(filesFound).
		FilesProcessed(filesProcessed).
		DiscoveryDone().
		Update(ctx); err != nil {
		logger.Warn("failed to update processing progress", "job_id", jobID, "error", err)
	}

	// 3. Auto-detect and import guide.xml
	xmltvFile := findXMLTVFile(lib.Path)
	if xmltvFile != "" {
		logger.Info("found XMLTV file, importing EPG", "path", xmltvFile)
		imported, err := importEPG(ctx, liveTVDeps, lib.ID, xmltvFile, logger)
		if err != nil {
			logger.Warn("failed to import EPG", "path", xmltvFile, "error", err)
		} else {
			logger.Info("EPG import complete", "library_id", lib.ID, "programs", imported)
		}

		// Persist the XMLTV path so subsequent manual/scheduled EPG scans use it
		if lib.MonitoringConfig == nil {
			lib.MonitoringConfig = &library.MonitoringConfig{}
		}
		lib.MonitoringConfig.XmltvURL = xmltvFile
		if err := deps.MediaRepos.Library.UpdateMonitoring(ctx, lib.ID, lib.MonitoringEnabled, lib.MonitoringConfig); err != nil {
			logger.Warn("failed to persist XMLTV path", "error", err)
		}
	}

	// 4. Complete the scan job
	now := time.Now()
	job := &scanner.ScanJob{
		ID:             jobID,
		LibraryID:      lib.ID,
		Status:         scanner.ScanStatusCompleted,
		FilesFound:     filesFound,
		FilesProcessed: filesProcessed,
		ErrorCount:     0,
		Progress:       100.0,
		CompletedAt:    &now,
		Phase:          scanner.ScanPhaseCompleted,
		DiscoveryDone:  true,
	}
	status.CompleteSafely(ctx, deps.StatusDeps(), job)

	logger.Info("live TV scan completed",
		"library_id", lib.ID,
		"m3u_files", filesFound,
		"channels_imported", totalChannels,
		"epg_imported", xmltvFile != "")
}

func findM3UFiles(path string) []string {
	entries, err := filepath.Glob(filepath.Join(path, "*.m3u"))
	if err != nil {
		return nil
	}
	if len(entries) == 0 {
		entries, err = filepath.Glob(filepath.Join(path, "*.m3u8"))
		if err != nil {
			return nil
		}
	}
	return entries
}

func findXMLTVFile(path string) string {
	entries, err := filepath.Glob(filepath.Join(path, "guide.xml"))
	if err != nil || len(entries) == 0 {
		return ""
	}
	return entries[0]
}

func importEPG(ctx context.Context, deps *LiveTVDeps, libraryID int64, xmltvPath string, logger *slog.Logger) (int, error) {
	f, err := os.Open(xmltvPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open XMLTV file: %w", err)
	}
	defer f.Close()

	result, err := infraLivetv.ParseXMLTV(f)
	if err != nil {
		return 0, fmt.Errorf("failed to parse XMLTV: %w", err)
	}

	channels, err := deps.ChannelRepo.ListByLibrary(ctx, libraryID)
	if err != nil {
		return 0, fmt.Errorf("failed to list channels: %w", err)
	}

	// Build mapping: XMLTV channel ID -> database channel ID
	epgToChannelID := make(map[string]int64)

	// 1. Load explicit mappings from database
	if deps.EPGQuerier != nil {
		mappings, err := deps.EPGQuerier.GetEPGMappingForLibrary(ctx, libraryID)
		if err == nil {
			for _, m := range mappings {
				epgToChannelID[m.XmltvChannelID] = m.ChannelID
			}
		}
	}

	// 2. Fall back to channel's EPGChannelID
	for _, ch := range channels {
		if ch.EPGChannelID != "" {
			if _, exists := epgToChannelID[ch.EPGChannelID]; !exists {
				epgToChannelID[ch.EPGChannelID] = ch.ID
			}
		}
	}

	// Debug: log XMLTV channel IDs that have programs but no matching M3U channel
	xmltvChannelsWithPrograms := make(map[string]bool)
	for _, p := range result.Programs {
		xmltvChannelsWithPrograms[p.ChannelID] = true
	}
	for xmltvID := range xmltvChannelsWithPrograms {
		if _, ok := epgToChannelID[xmltvID]; !ok {
			logger.Warn("XMLTV channel ID has no matching M3U channel (tvg-id)", "xmltv_id", xmltvID, "library_id", libraryID)
		}
	}

	programs := infraLivetv.ToDomainPrograms(result.Programs, epgToChannelID)

	if err := deps.ProgramRepo.DeleteByLibrary(ctx, libraryID); err != nil {
		return 0, fmt.Errorf("failed to clear old EPG data: %w", err)
	}

	if err := deps.ProgramRepo.BulkCreate(ctx, programs); err != nil {
		return 0, fmt.Errorf("failed to create EPG programs: %w", err)
	}

	return len(programs), nil
}
