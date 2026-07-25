package library

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mantonx/viewra/internal/application/library/scan"
	"github.com/mantonx/viewra/internal/application/library/scan/cleanup"
	"github.com/mantonx/viewra/internal/application/library/scan/discovery"
	"github.com/mantonx/viewra/internal/application/library/scan/execution"
	scanmedia "github.com/mantonx/viewra/internal/application/library/scan/media"
	"github.com/mantonx/viewra/internal/application/library/scan/processing"
	"github.com/mantonx/viewra/internal/application/library/scan/recovery"
	"github.com/mantonx/viewra/internal/application/library/scan/scanutil"
	"github.com/mantonx/viewra/internal/application/library/scan/status"
	domainImages "github.com/mantonx/viewra/internal/domain/images"
	"github.com/mantonx/viewra/internal/domain/library"
	"github.com/mantonx/viewra/internal/domain/media"
	"github.com/mantonx/viewra/internal/domain/scanner"
	"github.com/mantonx/viewra/internal/infrastructure/events"
	"github.com/mantonx/viewra/internal/infrastructure/filesystem"
	"github.com/mantonx/viewra/internal/infrastructure/system"
)

// =============================================================================
// ScanLibraryUseCase - Main Struct
// =============================================================================

// ScanLibraryUseCase handles the business logic for scanning a library
type ScanLibraryUseCase struct {
	mediaRepos         *scan.MediaRepositories
	scanRepos          *scan.ScanRepositories
	imageRepo          domainImages.Repository
	imageCleanup       cleanup.ImageCleanupExecutor
	incrementalScanner *discovery.IncrementalScanner
	coordinator        *filesystem.Coordinator
	config             scan.Config
	systemProfile      *system.Profile
	logger             *slog.Logger

	// Enrichment - optional, if set, media is enqueued for enrichment after scanning
	enrichmentEnqueuer scanmedia.EnrichmentEnqueuer

	// Event bus - optional, if set, scan events are published for SSE streaming
	eventBus *events.Bus

	// Live TV - optional, only set for live_tv library scans
	liveTVDeps *execution.LiveTVDeps

	// Per-session deduplication
	processedArtists scanutil.AtomicDeduplicator
	processedShows   scanutil.AtomicDeduplicator

	// New dependencies for simplified scan flow
	processingQueue library.ProcessingQueue
	pendingIDsRepo  media.PendingIdentificationRepository
}

// NewScanLibraryUseCase creates a new instance of ScanLibraryUseCase
func NewScanLibraryUseCase(
	mediaRepos *scan.MediaRepositories,
	scanRepos *scan.ScanRepositories,
	imageRepo domainImages.Repository,
	imageCleanup cleanup.ImageCleanupExecutor,
	enrichmentEnqueuer scanmedia.EnrichmentEnqueuer,
	config scan.Config,
	systemProfile *system.Profile,
	logger *slog.Logger,
) *ScanLibraryUseCase {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	config = config.WithDefaults()

	incrementalScanner := discovery.NewIncrementalScanner(scanRepos.ScanState, logger)

	coordinatorConfig := filesystem.DefaultCoordinatorConfig()
	coordinatorConfig.Logger = logger
	coordinator := filesystem.NewCoordinator(coordinatorConfig)

	return &ScanLibraryUseCase{
		mediaRepos:         mediaRepos,
		scanRepos:          scanRepos,
		imageRepo:          imageRepo,
		imageCleanup:       imageCleanup,
		enrichmentEnqueuer: enrichmentEnqueuer,
		incrementalScanner: incrementalScanner,
		coordinator:        coordinator,
		config:             config,
		systemProfile:      systemProfile,
		logger:             logger,
	}
}

// NewScanLibraryUseCaseWithQueue creates a new instance with processing queue support
func NewScanLibraryUseCaseWithQueue(
	mediaRepos *scan.MediaRepositories,
	scanRepos *scan.ScanRepositories,
	imageRepo domainImages.Repository,
	imageCleanup cleanup.ImageCleanupExecutor,
	enrichmentEnqueuer scanmedia.EnrichmentEnqueuer,
	processingQueue library.ProcessingQueue,
	pendingIDsRepo media.PendingIdentificationRepository,
	config scan.Config,
	systemProfile *system.Profile,
	logger *slog.Logger,
) *ScanLibraryUseCase {
	uc := NewScanLibraryUseCase(mediaRepos, scanRepos, imageRepo, imageCleanup,
		enrichmentEnqueuer, config, systemProfile, logger)
	uc.processingQueue = processingQueue
	uc.pendingIDsRepo = pendingIDsRepo
	return uc
}

// SetEventBus sets the event bus for publishing scan events.
// This is optional - if not set, no events will be published.
func (uc *ScanLibraryUseCase) SetEventBus(bus *events.Bus) {
	uc.eventBus = bus
}

// SetLiveTVDeps sets the dependencies for scanning live TV libraries.
// This is optional - only needed if the server has live_tv libraries configured.
func (uc *ScanLibraryUseCase) SetLiveTVDeps(deps *execution.LiveTVDeps) {
	uc.liveTVDeps = deps
}

// =============================================================================
// Public API Methods
// =============================================================================

// StartScan initiates a new scan for a library
func (uc *ScanLibraryUseCase) StartScan(ctx context.Context, libraryID int64) (scan.StartScanResponse, error) {
	lib, err := uc.mediaRepos.Library.GetByID(ctx, libraryID)
	if err != nil {
		return scan.StartScanResponse{}, fmt.Errorf("failed to get library: %w", err)
	}

	running, err := uc.scanRepos.ScanJob.ListRunning(ctx)
	if err != nil {
		return scan.StartScanResponse{}, fmt.Errorf("failed to check running scans: %w", err)
	}
	for _, job := range running {
		if job.LibraryID == libraryID {
			return scan.StartScanResponse{}, scanner.ErrAlreadyRunning
		}
	}

	if uc.logger == nil {
		uc.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	var estimatedTotal int64
	previousScan, err := uc.scanRepos.ScanJob.GetLatestByLibrary(ctx, libraryID)
	if err == nil && previousScan != nil && previousScan.Status == scanner.ScanStatusCompleted {
		estimatedTotal = previousScan.FilesFound
		uc.logger.Info("Using previous scan for progress estimation",
			"library_id", libraryID,
			"previous_files_found", estimatedTotal)
	}

	job := &scanner.ScanJob{
		LibraryID:      libraryID,
		Status:         scanner.ScanStatusRunning,
		Progress:       0,
		FilesFound:     0,
		FilesProcessed: 0,
		BytesProcessed: 0,
		ErrorCount:     0,
		StartedAt:      time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Phase:          scanner.ScanPhaseDiscovering,
		EstimatedTotal: estimatedTotal,
		DiscoveryDone:  false,
	}

	if err := uc.scanRepos.ScanJob.Create(ctx, job); err != nil {
		return scan.StartScanResponse{}, fmt.Errorf("failed to create scan job: %w", err)
	}

	// Publish scan started event
	uc.publishScanStarted(job)

	uc.startScanBackground(job.ID, lib, "scan goroutine panicked")
	return scan.ToStartScanResponse(job), nil
}

// StartTargetedScan initiates a new scan for specific paths within a library.
// This is used by the monitoring system to scan only changed/new files.
// targetPaths are relative to the library root (e.g., "Movies/Action", "Movies/Drama/The Godfather.mkv").
// Returns interface{} to avoid circular dependencies with monitor package (actual type is scan.StartScanResponse).
func (uc *ScanLibraryUseCase) StartTargetedScan(ctx context.Context, libraryID int64, targetPaths []string) (interface{}, error) {
	if len(targetPaths) == 0 {
		return scan.StartScanResponse{}, fmt.Errorf("targetPaths cannot be empty")
	}

	lib, err := uc.mediaRepos.Library.GetByID(ctx, libraryID)
	if err != nil {
		return scan.StartScanResponse{}, fmt.Errorf("failed to get library: %w", err)
	}

	// Check if there's already a running scan for this library
	running, err := uc.scanRepos.ScanJob.ListRunning(ctx)
	if err != nil {
		return scan.StartScanResponse{}, fmt.Errorf("failed to check running scans: %w", err)
	}
	for _, job := range running {
		if job.LibraryID == libraryID {
			return scan.StartScanResponse{}, scanner.ErrAlreadyRunning
		}
	}

	if uc.logger == nil {
		uc.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	// For targeted scans, we don't use estimated total since we're only scanning specific paths
	job := &scanner.ScanJob{
		LibraryID:      libraryID,
		Status:         scanner.ScanStatusRunning,
		Progress:       0,
		FilesFound:     0,
		FilesProcessed: 0,
		BytesProcessed: 0,
		ErrorCount:     0,
		StartedAt:      time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Phase:          scanner.ScanPhaseDiscovering,
		EstimatedTotal: 0, // Don't estimate for targeted scans
		DiscoveryDone:  false,
		TargetPaths:    targetPaths, // Set the target paths for filtering
	}

	if err := uc.scanRepos.ScanJob.Create(ctx, job); err != nil {
		return scan.StartScanResponse{}, fmt.Errorf("failed to create scan job: %w", err)
	}

	uc.logger.Info("Starting targeted scan",
		"job_id", job.ID,
		"library_id", libraryID,
		"target_paths", targetPaths)

	// Publish scan started event
	uc.publishScanStarted(job)

	uc.startScanBackground(job.ID, lib, "targeted scan goroutine panicked")
	return scan.ToStartScanResponse(job), nil
}

// publishScanStarted publishes a scan.started event if EventBus is configured.
func (uc *ScanLibraryUseCase) publishScanStarted(job *scanner.ScanJob) {
	if uc.eventBus == nil {
		return
	}

	uc.eventBus.Publish(events.NewEvent(events.EventScanStarted, "scanner").
		WithLibraryID(job.LibraryID).
		WithData("job_id", job.ID).
		WithData("status", string(job.Status)).
		WithData("phase", string(job.Phase)).
		WithData("estimated_total", job.EstimatedTotal).
		Build())
}

// ResumeScan resumes a paused scan job
func (uc *ScanLibraryUseCase) ResumeScan(ctx context.Context, jobID int64) error {
	job, err := uc.scanRepos.ScanJob.GetByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get scan job: %w", err)
	}

	if job.Status != scanner.ScanStatusPaused {
		return fmt.Errorf("scan job is not paused (current status: %s)", job.Status)
	}

	lib, err := uc.mediaRepos.Library.GetByID(ctx, job.LibraryID)
	if err != nil {
		return fmt.Errorf("failed to get library: %w", err)
	}

	if err := uc.scanRepos.ScanJob.UpdateStatus(ctx, jobID, scanner.ScanStatusRunning); err != nil {
		return fmt.Errorf("failed to update scan status: %w", err)
	}

	uc.logger.Info("resuming scan from user request",
		"job_id", jobID,
		"library_id", lib.ID,
		"files_processed", job.FilesProcessed,
		"files_found", job.FilesFound)

	uc.startScanBackground(job.ID, lib, "resumed scan goroutine panicked")
	return nil
}

// ResumeStuckScans automatically resumes scans interrupted by server restart
func (uc *ScanLibraryUseCase) ResumeStuckScans(ctx context.Context) error {
	runningScans, err := uc.scanRepos.ScanJob.ListRunning(ctx)
	if err != nil {
		return fmt.Errorf("failed to get running scans: %w", err)
	}

	if len(runningScans) == 0 {
		uc.logger.Debug("no stuck scans found during startup")
		return nil
	}

	uc.logger.Info("found stuck scans, automatically resuming", "count", len(runningScans))
	for _, job := range runningScans {
		execution.HandleStuckScan(ctx, uc.executionDeps(), job, uc)
	}
	return nil
}

// GetProgress retrieves the current progress of a scan job
func (uc *ScanLibraryUseCase) GetProgress(ctx context.Context, jobID int64) (scan.ScanProgressResponse, error) {
	job, err := uc.scanRepos.ScanJob.GetByID(ctx, jobID)
	if err != nil {
		return scan.ScanProgressResponse{}, fmt.Errorf("failed to get scan job: %w", err)
	}
	return scan.ToScanProgressResponse(job), nil
}

// GetLatestScan retrieves the most recent scan for a library
func (uc *ScanLibraryUseCase) GetLatestScan(ctx context.Context, libraryID int64) (scan.ScanProgressResponse, error) {
	job, err := uc.scanRepos.ScanJob.GetLatestByLibrary(ctx, libraryID)
	if err != nil {
		return scan.ScanProgressResponse{}, fmt.Errorf("failed to get latest scan: %w", err)
	}
	return scan.ToScanProgressResponse(job), nil
}

// GetScanHistory retrieves scan history for a library
func (uc *ScanLibraryUseCase) GetScanHistory(ctx context.Context, libraryID int64, limit int32) (scan.ScanHistoryResponse, error) {
	jobs, err := uc.scanRepos.ScanJob.ListByLibrary(ctx, libraryID, limit)
	if err != nil {
		return scan.ScanHistoryResponse{}, fmt.Errorf("failed to get scan history: %w", err)
	}
	return scan.ToScanHistoryResponse(jobs), nil
}

// GetScanStatus retrieves the enriched scan status for a library
func (uc *ScanLibraryUseCase) GetScanStatus(ctx context.Context, libraryID int64) (*status.Result, error) {
	return status.GetScanStatus(ctx, uc.statusDeps(), libraryID)
}

// =============================================================================
// Dependency Bundles
// =============================================================================

func (uc *ScanLibraryUseCase) mediaDeps() *scanmedia.Deps {
	deps := &scanmedia.Deps{
		MediaRepos:         uc.mediaRepos,
		ScanRepos:          uc.scanRepos,
		EnrichmentEnqueuer: uc.enrichmentEnqueuer,
		ImageRepo:          uc.imageRepo,
		ProcessedArtists:   &uc.processedArtists,
		ProcessedShows:     &uc.processedShows,
		Coordinator:        uc.coordinator,
		Logger:             uc.logger,
	}
	// Only set Publisher if eventBus is non-nil to avoid interface nil pitfall
	// (a nil *Bus assigned to Publisher interface is not == nil)
	if uc.eventBus != nil {
		deps.Publisher = uc.eventBus
	}
	return deps
}

func (uc *ScanLibraryUseCase) processingDeps() *processing.Deps {
	return &processing.Deps{
		ScanRepos:      uc.scanRepos,
		MediaRepos:     uc.mediaRepos,
		MediaProcessor: uc,
		Coordinator:    uc.coordinator,
		Config:         &uc.config,
		SystemProfile:  uc.systemProfile,
		EventBus:       uc.eventBus,
		Logger:         uc.logger,
	}
}

func (uc *ScanLibraryUseCase) discoveryDeps() *discovery.Deps {
	return &discovery.Deps{
		ScanRepos:     uc.scanRepos,
		Config:        &uc.config,
		SystemProfile: uc.systemProfile,
		Coordinator:   uc.coordinator,
		Logger:        uc.logger,
		IsMediaFile:   scanutil.IsMediaFile,
		IncrScanner:   uc.incrementalScanner,
	}
}

func (uc *ScanLibraryUseCase) statusDeps() *status.Deps {
	return &status.Deps{
		ScanRepos: uc.scanRepos,
		EventBus:  uc.eventBus,
		Logger:    uc.logger,
	}
}

func (uc *ScanLibraryUseCase) cleanupDeps() *cleanup.Deps {
	return &cleanup.Deps{
		MediaRepos:   uc.mediaRepos,
		ImageRepo:    uc.imageRepo,
		ImageCleanup: uc.imageCleanup,
		Logger:       uc.logger,
		Publisher:    uc.eventBus, // For media.removed events
	}
}

func (uc *ScanLibraryUseCase) executionDeps() *execution.Deps {
	return &execution.Deps{
		ScanRepos:                 uc.scanRepos,
		MediaRepos:                uc.mediaRepos,
		Config:                    &uc.config,
		SystemProfile:             uc.systemProfile,
		Logger:                    uc.logger,
		DiscoveryDeps:             uc.discoveryDeps,
		ProcessingDeps:            uc.processingDeps,
		StatusDeps:                uc.statusDeps,
		CleanupDeps:               uc.cleanupDeps,
		ProgressUpdater:           uc,
		SessionInitializer:        uc,
		RecoverFromPanic:          uc.recoverFromPanic,
		RecoverFromPanicWithError: uc.recoverFromPanicWithError,
		HasImageCleanup:           func() bool { return uc.imageRepo != nil && uc.imageCleanup != nil },
	}
}

// =============================================================================
// Interface Implementations
// =============================================================================

var _ processing.MediaProcessor = (*ScanLibraryUseCase)(nil)
var _ execution.ProgressUpdater = (*ScanLibraryUseCase)(nil)
var _ execution.SessionInitializer = (*ScanLibraryUseCase)(nil)
var _ execution.BackgroundStarter = (*ScanLibraryUseCase)(nil)

func (uc *ScanLibraryUseCase) ProcessMovie(ctx context.Context, libraryID int64, result *scanner.ScanResult, checkpoint *scanner.ScanCheckpoint, cache *sync.Map) (*int64, error) {
	return scanmedia.ProcessMovie(ctx, uc.mediaDeps(), libraryID, result, checkpoint, cache)
}

func (uc *ScanLibraryUseCase) ProcessTVEpisode(ctx context.Context, libraryID int64, result *scanner.ScanResult, checkpoint *scanner.ScanCheckpoint, cache *sync.Map) (*int64, error) {
	return scanmedia.ProcessTVEpisode(ctx, uc.mediaDeps(), libraryID, result, checkpoint, cache)
}

func (uc *ScanLibraryUseCase) ProcessMusicTrack(ctx context.Context, libraryID int64, result *scanner.ScanResult, checkpoint *scanner.ScanCheckpoint, cache *sync.Map) (*int64, error) {
	return scanmedia.ProcessMusicTrack(ctx, uc.mediaDeps(), libraryID, result, checkpoint, cache)
}

func (uc *ScanLibraryUseCase) NewProgressUpdate(jobID int64) *status.ProgressUpdate {
	return status.NewProgressUpdate(uc.scanRepos.ScanJob, uc.logger, jobID)
}

func (uc *ScanLibraryUseCase) InitializeScanSession(ctx context.Context, lib *library.Library) {
	uc.processedArtists.Reset()
	uc.processedShows.Reset()

	if uc.systemProfile != nil {
		uc.systemProfile.UpdateForLibraryPath(ctx, lib.Path)
		uc.logger.Info("detected storage for library",
			"library_path", lib.Path,
			"storage_type", uc.systemProfile.Storage.Type,
			"is_remote", uc.systemProfile.Storage.IsRemote)
	}
}

func (uc *ScanLibraryUseCase) StartScanBackground(jobID int64, libraryID int64, panicContext string) {
	lib, err := uc.mediaRepos.Library.GetByID(context.Background(), libraryID)
	if err != nil {
		uc.logger.Error("failed to get library for background scan", "library_id", libraryID, "error", err)
		return
	}
	uc.startScanBackground(jobID, lib, panicContext)
}

// =============================================================================
// Internal Scan Logic
// =============================================================================

func (uc *ScanLibraryUseCase) startScanBackground(jobID int64, lib *library.Library, panicContext string) {
	scanCtx, cancel := context.WithTimeout(context.Background(), uc.config.Timeout)
	go func() {
		defer cancel()
		defer uc.recoverFromPanic(jobID, lib.ID, panicContext)
		uc.runScan(scanCtx, jobID, lib)
	}()
}

func (uc *ScanLibraryUseCase) runScan(ctx context.Context, jobID int64, lib *library.Library) {
	uc.InitializeScanSession(ctx, lib)

	currentJob, err := uc.scanRepos.ScanJob.GetByID(ctx, jobID)
	if err != nil {
		uc.logger.Error("failed to get scan job", "job_id", jobID, "error", err)
		status.CompleteWithError(ctx, uc.statusDeps(), jobID, err)
		return
	}

	params := &execution.RunScanParams{JobID: jobID, Lib: lib}
	if execution.CanResumeFromCheckpoints(ctx, uc.executionDeps(), params, currentJob) {
		return
	}

	uc.logger.Info("starting fresh scan", "library_id", lib.ID, "type", lib.Type)
	if lib.Type == library.LibraryTypeLiveTV {
		if uc.liveTVDeps == nil {
			status.CompleteWithError(ctx, uc.statusDeps(), jobID, fmt.Errorf("live TV dependencies not configured"))
			return
		}
		execution.RunLiveTVScan(ctx, uc.executionDeps(), uc.liveTVDeps, jobID, lib)
		return
	}
	execution.RunFreshScan(ctx, uc.executionDeps(), jobID, lib)
}

// =============================================================================
// Panic Recovery
// =============================================================================

func (uc *ScanLibraryUseCase) recoverFromPanic(jobID, libraryID int64, description string) {
	recovery.RecoverFromPanic(uc.logger, uc.scanRepos.ScanJob, jobID, libraryID, description)
}

func (uc *ScanLibraryUseCase) recoverFromPanicWithError(jobID, libraryID int64, description string, errChan chan<- error) {
	recovery.RecoverFromPanicWithError(uc.logger, jobID, libraryID, description, errChan)
}

// =============================================================================
// Simplified Scan Flow Methods
// =============================================================================

// StartSimplifiedScan initiates a simplified scan that only discovers files and enqueues them
// for background processing. This is the new scan flow that replaces the full scan.
func (uc *ScanLibraryUseCase) StartSimplifiedScan(ctx context.Context, libraryID int64) (scan.StartScanResponse, error) {
	if uc.processingQueue == nil {
		return scan.StartScanResponse{}, fmt.Errorf("processing queue not configured")
	}

	lib, err := uc.mediaRepos.Library.GetByID(ctx, libraryID)
	if err != nil {
		return scan.StartScanResponse{}, fmt.Errorf("failed to get library: %w", err)
	}

	// Check for running scans
	running, err := uc.scanRepos.ScanJob.ListRunning(ctx)
	if err != nil {
		return scan.StartScanResponse{}, fmt.Errorf("failed to check running scans: %w", err)
	}
	for _, job := range running {
		if job.LibraryID == libraryID {
			return scan.StartScanResponse{}, scanner.ErrAlreadyRunning
		}
	}

	if uc.logger == nil {
		uc.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	// Get previous scan for progress estimation
	var estimatedTotal int64
	previousScan, err := uc.scanRepos.ScanJob.GetLatestByLibrary(ctx, libraryID)
	if err == nil && previousScan != nil && previousScan.Status == scanner.ScanStatusCompleted {
		estimatedTotal = previousScan.FilesFound
	}

	job := &scanner.ScanJob{
		LibraryID:      libraryID,
		Status:         scanner.ScanStatusRunning,
		Progress:       0,
		FilesFound:     0,
		FilesProcessed: 0,
		BytesProcessed: 0,
		ErrorCount:     0,
		StartedAt:      time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Phase:          scanner.ScanPhaseDiscovering,
		EstimatedTotal: estimatedTotal,
		DiscoveryDone:  false,
	}

	if err := uc.scanRepos.ScanJob.Create(ctx, job); err != nil {
		return scan.StartScanResponse{}, fmt.Errorf("failed to create scan job: %w", err)
	}

	uc.publishScanStarted(job)

	// Start simplified scan in background
	go func() {
		defer uc.recoverFromPanic(job.ID, lib.ID, "simplified scan panicked")
		uc.runSimplifiedScan(context.Background(), job.ID, lib)
	}()

	return scan.ToStartScanResponse(job), nil
}

// runSimplifiedScan runs the simplified scan flow
func (uc *ScanLibraryUseCase) runSimplifiedScan(ctx context.Context, jobID int64, lib *library.Library) {
	uc.InitializeScanSession(ctx, lib)

	// Step 1: Discover files
	newFiles, err := uc.discoverFiles(ctx, jobID, lib)
	if err != nil {
		uc.logger.Error("failed to discover files", "error", err)
		status.CompleteWithError(ctx, uc.statusDeps(), jobID, err)
		return
	}

	if len(newFiles) > 0 {
		// Step 2a: New files found -> promote pending IDs
		if err := uc.promotePendingIDsForFiles(ctx, lib.ID, newFiles); err != nil {
			uc.logger.Error("failed to promote pending IDs", "error", err)
			// Continue - not fatal
		}

		// Enqueue new files for processing
		if err := uc.enqueueFilesForProcessing(ctx, lib.ID, newFiles); err != nil {
			uc.logger.Error("failed to enqueue files", "error", err)
			status.CompleteWithError(ctx, uc.statusDeps(), jobID, err)
			return
		}
	} else {
		// Step 2b: No new files -> detect incomplete items
		incomplete, err := uc.detectIncompleteItems(ctx, lib.ID)
		if err != nil {
			uc.logger.Error("failed to detect incomplete items", "error", err)
			status.CompleteWithError(ctx, uc.statusDeps(), jobID, err)
			return
		}

		if len(incomplete) > 0 {
			// Enqueue incomplete items for re-processing
			if err := uc.enqueueIncompleteForProcessing(ctx, lib.ID, incomplete); err != nil {
				uc.logger.Error("failed to enqueue incomplete items", "error", err)
				status.CompleteWithError(ctx, uc.statusDeps(), jobID, err)
				return
			}
		}
	}

	// Step 3: Complete
	job := &scanner.ScanJob{
		ID:             jobID,
		Status:         scanner.ScanStatusCompleted,
		Phase:          scanner.ScanPhaseCompleted,
		CompletedAt:    &[]time.Time{time.Now()}[0],
		DiscoveryDone:  true,
	}
	status.CompleteSafely(ctx, uc.statusDeps(), job)
}

// discoverFiles discovers media files in the library
func (uc *ScanLibraryUseCase) discoverFiles(ctx context.Context, jobID int64, lib *library.Library) ([]string, error) {
	// Get current job
	currentJob, err := uc.scanRepos.ScanJob.GetByID(ctx, jobID)
	if err != nil {
		return nil, err
	}

	// Create discovery context
	dDeps := uc.discoveryDeps()
	walker := discovery.CreateWalker(dDeps)
	dctx := discovery.NewContext(jobID, lib, currentJob, walker)

	// Phase 1: Count files
	discovery.PhaseCountFiles(ctx, dctx, dDeps)

	// Phase 2: Walk directory
	discoveredFiles, err := discovery.PhaseWalkDirectory(ctx, dctx, dDeps, nil)
	if err != nil {
		return nil, err
	}

	// Phase 3: Determine changes (returns diff with new files)
	diff := discovery.PhaseDetermineChanges(ctx, dctx, dDeps, discoveredFiles)
	if diff == nil {
		// No changes
		return nil, nil
	}

	// Return new file paths
	var newFiles []string
	for _, file := range diff.NewFiles {
		newFiles = append(newFiles, file.Path)
	}
	for _, file := range diff.ModifiedFiles {
		newFiles = append(newFiles, file.Path)
	}

	return newFiles, nil
}

// promotePendingIDsForFiles promotes pending IDs for discovered files
func (uc *ScanLibraryUseCase) promotePendingIDsForFiles(ctx context.Context, libraryID int64, filePaths []string) error {
	if uc.pendingIDsRepo == nil {
		return nil
	}

	// Get pending IDs for these files
	pendingIDs, err := uc.pendingIDsRepo.GetByLibraryAndPaths(ctx, libraryID, filePaths)
	if err != nil {
		return err
	}

	if len(pendingIDs) == 0 {
		return nil
	}

	// Group by file path
	byPath := make(map[string][]*media.PendingIdentification)
	for _, id := range pendingIDs {
		byPath[id.FilePath] = append(byPath[id.FilePath], id)
	}

	// For each file, we need to find or create the media record to get the media ID
	// This will be done by the background worker after FFprobe processing
	// For now, we just keep the pending IDs in the table

	uc.logger.Info("promoted pending IDs for files",
		"library_id", libraryID,
		"file_count", len(byPath),
		"total_ids", len(pendingIDs))

	return nil
}

// enqueueFilesForProcessing adds files to the processing queue
func (uc *ScanLibraryUseCase) enqueueFilesForProcessing(ctx context.Context, libraryID int64, filePaths []string) error {
	items := make([]*library.ProcessingItem, 0, len(filePaths))
	for _, filePath := range filePaths {
		items = append(items, &library.ProcessingItem{
			LibraryID: libraryID,
			FilePath:  filePath,
			Status:    library.ProcessingStatusPending,
		})
	}

	return uc.processingQueue.EnqueueBatch(ctx, items)
}

// detectIncompleteItems finds items missing images OR NFO that aren't already in queue
func (uc *ScanLibraryUseCase) detectIncompleteItems(ctx context.Context, libraryID int64) ([]int64, error) {
	// Get all media items in library
	mediaItems, err := uc.mediaRepos.Media.ListByLibrary(ctx, libraryID)
	if err != nil {
		return nil, err
	}

	var incomplete []int64
	for _, item := range mediaItems {
		// Check if already in processing queue
		inQueue, err := uc.processingQueue.ExistsInQueue(ctx, libraryID, item.FilePath)
		if err != nil {
			uc.logger.Warn("failed to check queue", "file_path", item.FilePath, "error", err)
			continue
		}
		if inQueue {
			continue
		}

		// Check images
		hasImages, err := uc.checkHasImages(ctx, item.ID)
		if err != nil {
			uc.logger.Warn("failed to check images", "media_id", item.ID, "error", err)
			continue
		}

		// Check NFO file
		hasNFO := uc.checkHasNFO(item.FilePath)

		if !hasImages || !hasNFO {
			incomplete = append(incomplete, item.ID)
		}
	}

	return incomplete, nil
}

// checkHasImages checks if a media item has images
func (uc *ScanLibraryUseCase) checkHasImages(ctx context.Context, mediaID int64) (bool, error) {
	if uc.imageRepo == nil {
		return true, nil // If no image repo, assume images exist
	}

	images, err := uc.imageRepo.GetByMediaID(ctx, int(mediaID))
	if err != nil {
		return false, err
	}

	return len(images) > 0, nil
}

// checkHasNFO checks if a media file has an NFO file
func (uc *ScanLibraryUseCase) checkHasNFO(mediaPath string) bool {
	nfoPath := mediaPathToNFOPath(mediaPath)
	_, err := os.Stat(nfoPath)
	return err == nil
}

// mediaPathToNFOPath converts a media file path to an NFO file path
func mediaPathToNFOPath(mediaPath string) string {
	dir := filepath.Dir(mediaPath)
	base := strings.TrimSuffix(filepath.Base(mediaPath), filepath.Ext(mediaPath))
	return filepath.Join(dir, base+".nfo")
}

// enqueueIncompleteForProcessing adds incomplete items to the processing queue
func (uc *ScanLibraryUseCase) enqueueIncompleteForProcessing(ctx context.Context, libraryID int64, mediaIDs []int64) error {
	items := make([]*library.ProcessingItem, 0, len(mediaIDs))
	for _, mediaID := range mediaIDs {
		// Get media item to find file path
		mediaItem, err := uc.mediaRepos.Media.GetByID(ctx, mediaID)
		if err != nil {
			uc.logger.Warn("failed to get media item", "media_id", mediaID, "error", err)
			continue
		}

		items = append(items, &library.ProcessingItem{
			LibraryID: libraryID,
			FilePath:  mediaItem.FilePath,
			MediaID:   &mediaID,
			Status:    library.ProcessingStatusPending,
		})
	}

	return uc.processingQueue.EnqueueBatch(ctx, items)
}
