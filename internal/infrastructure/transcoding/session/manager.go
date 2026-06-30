package session

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	domainevents "github.com/mantonx/viewra/internal/domain/events"
	"github.com/mantonx/viewra/internal/infrastructure/ffmpeg/hls"
	"github.com/mantonx/viewra/internal/infrastructure/transcoding/logging"
	"github.com/mantonx/viewra/internal/infrastructure/transcoding/profile"
	"github.com/mantonx/viewra/internal/infrastructure/transcoding/videoinfo"
)

// ConfigProvider provides transcoding configuration.
// This minimal interface allows the SessionManager to work with dynamic config
// from the application layer without creating import cycles.
type ConfigProvider interface {
	GetConfig(ctx context.Context) *Config
}

// Manager manages active transcode sessions.
// Handles session lifecycle, cleanup, and restart logic for seeking.
type Manager struct {
	sessions        sync.Map // map[string]*TranscodeSession (key: "mediaID_quality")
	sessionMu       sync.Map // map[string]*sync.Mutex - per-session creation locks
	logger          *slog.Logger
	config          *Config
	hwAccel         string // Default hardware acceleration (from ManagerConfig)
	hwDevice        string // Default hardware device (from ManagerConfig)
	configProvider  ConfigProvider
	fallbackManager *hls.HardwareFallbackManager
	logStore        *logging.FFmpegLogStore
	publisher       domainevents.Publisher // Event publisher for transcode events (optional)

	// Live stream session tracking
	liveMu        sync.Mutex
	pausedSession *TranscodeSession // Currently paused live session (only one allowed)

	outputBaseDir string
}

// ManagerConfig contains all configuration needed for the session manager.
type ManagerConfig struct {
	FFmpegPaths                *hls.Paths
	HardwareAccel              string // "none", "vaapi", "nvenc", "qsv", "videotoolbox"
	HardwareDevice             string
	OutputBaseDir              string
	MinFreeDiskGB              int64
	MaxCPUPercent              int
	MaxMemoryMB                int
	ProcessGroupKill           bool
	ToneMappingEnabled         bool
	ToneMappingAlgorithm       string
	ToneMappingBackend         string
	LibPlaceboPeakDetect       bool
	LibPlaceboContrastRecovery float64
}

// NewManager creates a new session manager.
func NewManager(config *ManagerConfig, logger *slog.Logger) *Manager {
	if config == nil {
		config = &ManagerConfig{
			HardwareAccel:      "none",
			ToneMappingEnabled: true,
		}
	}

	sessionConfig := &Config{
		FFmpegPaths:                config.FFmpegPaths,
		MaxMemoryMB:                config.MaxMemoryMB,
		ToneMappingEnabled:         config.ToneMappingEnabled,
		ToneMappingAlgorithm:       config.ToneMappingAlgorithm,
		ToneMappingBackend:         config.ToneMappingBackend,
		LibPlaceboPeakDetect:       config.LibPlaceboPeakDetect,
		LibPlaceboContrastRecovery: config.LibPlaceboContrastRecovery,
	}

	// Convert to hls config for fallback manager
	hlsConfig := &hls.Config{
		HardwareAccel: hls.HardwareAccel(config.HardwareAccel),
	}
	if config.FFmpegPaths != nil {
		hlsConfig.FFmpegPath = config.FFmpegPaths.FFmpeg
		hlsConfig.FFmpegLibPath = config.FFmpegPaths.LibPath
	}

	mgr := &Manager{
		logger:          logger,
		config:          sessionConfig,
		hwAccel:         config.HardwareAccel,
		hwDevice:        config.HardwareDevice,
		fallbackManager: hls.NewHardwareFallbackManager(hlsConfig, logger),
		outputBaseDir:   config.OutputBaseDir,
	}

	// Verify hardware acceleration is available
	hwAccel := hls.HardwareAccel(config.HardwareAccel)
	if hwAccel != hls.AccelNone {
		if err := mgr.fallbackManager.VerifyHardwareAvailability(hwAccel); err != nil {
			logger.Warn("Hardware acceleration verification failed, falling back to software",
				"hardware", config.HardwareAccel,
				"error", err,
			)
		}
	}

	return mgr
}

// SetConfigProvider sets a dynamic config provider for runtime settings.
func (m *Manager) SetConfigProvider(provider ConfigProvider) {
	m.configProvider = provider
}

// SetLogStore sets the FFmpeg log store for persistent log capture.
func (m *Manager) SetLogStore(store *logging.FFmpegLogStore) {
	m.logStore = store
}

// GetLogStore returns the FFmpeg log store (may be nil if not configured).
func (m *Manager) GetLogStore() *logging.FFmpegLogStore {
	return m.logStore
}

// SetPublisher sets the event publisher for transcode lifecycle events.
// If set, the manager will publish transcode.started, transcode.completed, etc.
func (m *Manager) SetPublisher(pub domainevents.Publisher) {
	m.publisher = pub
}

// getEffectiveConfig returns the current configuration, using provider if available.
func (m *Manager) getEffectiveConfig(ctx context.Context) *Config {
	if m.configProvider != nil {
		return m.configProvider.GetConfig(ctx)
	}
	return m.config
}

// getSessionMutex returns a mutex for the given session key.
func (m *Manager) getSessionMutex(key string) *sync.Mutex {
	mu := &sync.Mutex{}
	actual, _ := m.sessionMu.LoadOrStore(key, mu)
	return actual.(*sync.Mutex)
}

// GetOrCreateSessionParams contains parameters for GetOrCreateSession.
type GetOrCreateSessionParams struct {
	MediaID               int64
	Quality               string
	StartPosition         float64
	AudioTrackIndex       int
	InputPath             string
	Profile               *profile.AdaptiveProfile
	Strategy              string
	StrategyDisplay       string // Human-readable strategy name
	StrategyReason        string // Detailed reason for the strategy decision
	OutputDir             string
	VideoInfo             *videoinfo.VideoInfo
	ClientSupportedCodecs []string
	HWAccel               string
	HWDevice              string
}

// LiveStreamSessionParams contains parameters for live stream sessions.
type LiveStreamSessionParams struct {
	LibraryID        int64
	ChannelID        int64
	InputURL         string
	OutputDir        string
	Profile          *profile.AdaptiveProfile
	HWAccel          string
	HWDevice         string
	MaxDVRSegments   int           // Maximum segments to retain when paused (default: 1800 = 60 min at 2s)
	MaxDVRDuration   time.Duration // Maximum DVR duration (default: 60 min)
	MinFreeDiskPct   float64       // Minimum free disk percentage (default: 10.0)
}

// LiveStreamState represents the state of a live stream session.
type LiveStreamState int

const (
	LiveStatePlaying LiveStreamState = iota
	LiveStatePaused
	LiveStateEnded
)

// GetOrCreateSession returns an existing session or creates a new one.
func (m *Manager) GetOrCreateSession(params GetOrCreateSessionParams) (*TranscodeSession, error) {
	// Use manager's default HW settings if not specified in params
	hwAccel := params.HWAccel
	if hwAccel == "" {
		hwAccel = m.hwAccel
	}
	hwDevice := params.HWDevice
	if hwDevice == "" {
		hwDevice = m.hwDevice
	}

	key := sessionKey(params.MediaID, params.Quality, params.AudioTrackIndex)

	// Stop all sessions for OTHER media to prevent resource hogging
	m.stopOtherMediaSessions(params.MediaID)

	// Stop sessions for the SAME media but DIFFERENT qualities to prevent manifest conflicts
	// When quality changes, the old session would still be writing to its output dir
	// This causes HLS.js to receive conflicting playlists with mismatched media sequences
	m.StopOtherQualitySessions(params.MediaID, params.Quality)

	// Acquire per-key mutex to prevent race conditions
	mu := m.getSessionMutex(key)
	mu.Lock()
	defer mu.Unlock()

	// Check for existing session
	m.logger.Debug("Checking for existing session",
		"key", key,
		"media_id", params.MediaID,
		"quality", params.Quality,
		"start_position", params.StartPosition)

	if existing, ok := m.sessions.Load(key); ok {
		session := existing.(*TranscodeSession)

		// Determine reuse window based on strategy.
		// Remux strategies have fast startup (~1-3s), so we can afford a larger window
		// to reduce unnecessary session restarts during typical navigation patterns.
		// Transcode has slow startup (~10-30s), so use a smaller window to avoid
		// long waits when the user seeks far ahead.
		reuseWindowSec := 30.0 // Default for transcode
		isRemuxStrategy := params.Strategy == "remux" || params.Strategy == "remux_audio" || params.Strategy == "remux_hevc"
		if isRemuxStrategy {
			reuseWindowSec = 60.0 // Larger window for fast-startup remux
		}

		positionDiff := params.StartPosition - session.StartPosition
		canReuse := (positionDiff >= 0 && positionDiff <= reuseWindowSec)

		if canReuse {
			session.UpdateLastAccessed()

			m.logger.Debug("Reusing existing transcode session",
				"session_id", session.ID,
				"media_id", params.MediaID,
				"quality", params.Quality,
				"session_start", session.StartPosition,
				"requested_start", params.StartPosition,
				"position_diff", positionDiff,
				"reuse_window_sec", reuseWindowSec)

			return session, nil
		}

		// Position is outside reusable range - need to restart session
		m.logger.Info("Restarting session for seek operation",
			"session_id", session.ID,
			"media_id", params.MediaID,
			"quality", params.Quality,
			"session_start", session.StartPosition,
			"requested_start", params.StartPosition,
			"position_diff", positionDiff,
			"reuse_window_sec", reuseWindowSec)

		session.Stop()
		m.sessions.Delete(key)

		// Close log writer for old session
		if m.logStore != nil {
			m.logStore.CloseLogWriter(session.ID)
		}

		// Clean up old output directory before creating new session
		// This prevents stale segments from interfering with the new session
		m.cleanupOutputDir(session.OutputDir, session.ID)
	}

	// No existing session found - create new one
	m.logger.Info("Creating new transcode session (no existing session for key)",
		"key", key,
		"media_id", params.MediaID,
		"quality", params.Quality,
		"start_position", params.StartPosition)

	session := NewTranscodeSession(
		params.MediaID,
		params.Quality,
		params.StartPosition,
		params.AudioTrackIndex,
		params.OutputDir,
		m.logger,
		nil,
	)

	// Create log writer for this session
	if m.logStore != nil {
		logWriter, err := m.logStore.CreateLogWriter(session.ID, params.MediaID, params.Quality)
		if err != nil {
			m.logger.Warn("Failed to create FFmpeg log writer",
				"session_id", session.ID,
				"error", err)
		} else {
			session.SetLogWriter(logWriter)
		}
	}

	// Set event publisher for transcode lifecycle events
	if m.publisher != nil {
		session.SetPublisher(m.publisher)
	}

	// Get effective config
	effectiveConfig := m.getEffectiveConfig(context.Background())

	// Convert videoinfo.VideoInfo to hls.VideoInfo
	var videoInfo *hls.VideoInfo
	if params.VideoInfo != nil {
		videoInfo = convertProbeToHLSVideoInfo(params.VideoInfo)
	}

	// Start FFmpeg process
	startParams := StartParams{
		InputPath:             params.InputPath,
		Profile:               params.Profile,
		Strategy:              params.Strategy,
		HWAccel:               hwAccel,
		HWDevice:              hwDevice,
		VideoInfo:             videoInfo,
		Config:                effectiveConfig,
		ClientSupportedCodecs: params.ClientSupportedCodecs,
	}

	if err := session.Start(startParams); err != nil {
		// Check if this is a hardware error and fallback if needed
		hwAccelType := hls.HardwareAccel(hwAccel)
		if m.fallbackManager.RecordFailure(hwAccelType, err) {
			m.logger.Info("Retrying with fallback acceleration",
				"from", hwAccel,
				"to", m.fallbackManager.GetCurrentAccel(),
			)
			startParams.HWAccel = string(m.fallbackManager.GetCurrentAccel())
			if err := session.Start(startParams); err != nil {
				return nil, fmt.Errorf("failed to start transcode session after fallback: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to start transcode session: %w", err)
		}
	}

	// Store session
	m.sessions.Store(key, session)

	// Publish transcode.started event
	if m.publisher != nil {
		m.publisher.Publish(domainevents.NewEvent(domainevents.EventTranscodeStarted, "session-manager").
			WithMediaID(params.MediaID).
			WithData("session_id", session.ID).
			WithData("quality", params.Quality).
			WithData("strategy", params.Strategy).
			WithData("strategy_display", params.StrategyDisplay).
			WithData("strategy_reason", params.StrategyReason).
			WithData("hw_accel", startParams.HWAccel).
			WithData("start_position", params.StartPosition).
			Build())
	}

	m.logger.Debug("Created new transcode session",
		"session_id", session.ID,
		"media_id", params.MediaID,
		"quality", params.Quality,
		"start_position", params.StartPosition)

	return session, nil
}

// GetOrCreateLiveStreamSession creates or returns an existing live stream session.
// Only one live session can be paused at a time globally (returns 409 if another is paused).
func (m *Manager) GetOrCreateLiveStreamSession(params LiveStreamSessionParams) (*TranscodeSession, error) {
	// Use manager's default HW settings if not specified
	hwAccel := params.HWAccel
	if hwAccel == "" {
		hwAccel = m.hwAccel
	}
	hwDevice := params.HWDevice
	if hwDevice == "" {
		hwDevice = m.hwDevice
	}

	// Set defaults
	maxDVRSegments := params.MaxDVRSegments
	if maxDVRSegments <= 0 {
		maxDVRSegments = 1800 // 60 minutes at 2s segments
	}
	maxDVRDuration := params.MaxDVRDuration
	if maxDVRDuration <= 0 {
		maxDVRDuration = 60 * time.Minute
	}
	minFreeDiskPct := params.MinFreeDiskPct
	if minFreeDiskPct <= 0 {
		minFreeDiskPct = 10.0
	}

	key := liveStreamKey(params.LibraryID, params.ChannelID)

	// Stop any existing session for this channel
	m.liveMu.Lock()
	if existing, ok := m.sessions.Load(key); ok {
		session := existing.(*TranscodeSession)
		session.Stop()
		m.sessions.Delete(key)
		if m.logStore != nil {
			m.logStore.CloseLogWriter(session.ID)
		}
		m.cleanupOutputDir(session.OutputDir, session.ID)
	}
	m.liveMu.Unlock()

	// Acquire per-key mutex to prevent race conditions
	mu := m.getSessionMutex(key)
	mu.Lock()
	defer mu.Unlock()

	// Check for existing session (double-check after lock)
	if existing, ok := m.sessions.Load(key); ok {
		session := existing.(*TranscodeSession)
		session.UpdateLastAccessed()
		return session, nil
	}

	// Create new live stream session
	m.logger.Info("Creating new live stream session",
		"key", key,
		"library_id", params.LibraryID,
		"channel_id", params.ChannelID,
		"input_url", params.InputURL)

	outputDir := params.OutputDir
	if outputDir == "" {
		outputDir = m.outputBaseDir
	}

	session := NewTranscodeSession(
		params.LibraryID,  // Use libraryID as mediaID for key
		"live",            // Fixed quality "live"
		0.0,               // Start position
		0,                 // Audio track index
		outputDir,         // Base output dir
		m.logger,
		nil,
	)

	// Override output directory for live stream
	session.OutputDir = filepath.Join(outputDir, fmt.Sprintf("livetv_%d_%d", params.LibraryID, params.ChannelID))
	session.ManifestPath = filepath.Join(session.OutputDir, "playlist.m3u8")
	session.maxDVRSegments = maxDVRSegments
	session.minFreeDiskPct = minFreeDiskPct

	// Create log writer for this session
	if m.logStore != nil {
		logWriter, err := m.logStore.CreateLogWriter(session.ID, params.LibraryID, "live")
		if err != nil {
			m.logger.Warn("Failed to create FFmpeg log writer",
				"session_id", session.ID,
				"error", err)
		} else {
			session.SetLogWriter(logWriter)
		}
	}

	// Set event publisher
	if m.publisher != nil {
		session.SetPublisher(m.publisher)
	}

	// Get effective config
	effectiveConfig := m.getEffectiveConfig(context.Background())

	// For RTSP/SAT>IP sources, use a FIFO to buffer initial TS data.
	// The RTSP handler with satip_raw consumes initial TS bytes during probing,
	// which can prevent the H.264 decoder from receiving SPS/PPS. The FIFO lets
	// the mpegts demuxer find parameter sets from the stream start.
	inputURL := params.InputURL
	isRTSP := strings.HasPrefix(inputURL, "rtsp://")

	if isRTSP {
		if err := os.MkdirAll(session.OutputDir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create output directory for FIFO: %w", err)
		}
		fifoPath := filepath.Join(session.OutputDir, "livepipe.fifo")
		os.Remove(fifoPath)
		if err := syscall.Mkfifo(fifoPath, 0o666); err != nil {
			return nil, fmt.Errorf("failed to create FIFO for RTSP stream: %w", err)
		}
		session.fifoPath = fifoPath

		// Producer: captures RTSP stream as raw MPEG-TS to the FIFO
		producerArgs := []string{
			"-rtsp_flags", "satip_raw",
			"-timeout", "5000000",
			"-i", params.InputURL,
			"-c", "copy",
			"-f", "mpegts",
			"-y", fifoPath,
		}
		producerCmd := createFFmpegCommand(context.Background(), producerArgs, nil, m.logger)
		session.liveProducerCmd = producerCmd
		if err := producerCmd.Start(); err != nil {
			os.Remove(fifoPath)
			return nil, fmt.Errorf("failed to start RTSP producer: %w", err)
		}

		// Wait for initial TS data to buffer in the FIFO, ensuring SPS/PPS arrive
		// before the consumer's mpegts demuxer initializes the decoder.
		time.Sleep(2 * time.Second)

		inputURL = fifoPath
	}

	// Build FFmpeg input args.
	// -timeout is only valid for network protocols, so omit for FIFO input.
	inputArgs := []string{
		"-fflags", "+genpts",
		"-avoid_negative_ts", "make_zero",
		"-use_wallclock_as_timestamps", "1",
	}

	if !isRTSP {
		inputArgs = append([]string{"-timeout", "5000000"}, inputArgs...)
		// For non-FIFO inputs (e.g., local files), add -re to pace at native rate.
		inputArgs = append(inputArgs, "-re")
	}

	// Build output args for HLS with DVR
	outputArgs := []string{
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-crf", "23",
		"-b:v", "5000000",
		"-maxrate", "5500000",
		"-bufsize", "10000000",
		"-g", "48",
		"-keyint_min", "24",
		"-sc_threshold", "0",
		"-r", "24",
		"-c:a", "aac",
		"-b:a", "128000",
		"-ac", "2",
		"-ar", "48000",
		"-f", "hls",
		"-hls_time", "2",
		"-hls_list_size", "1800",
		"-hls_flags", "program_date_time+append_list+delete_segments",
		"-hls_segment_filename", "seg_%06d.ts",
		"-hls_playlist_type", "event",
		"-use_wallclock_as_timestamps", "1",
		"-strftime_mkdir", "1",
	}

	// Build complete FFmpeg command
	// Input args → -i inputURL → output args → output playlist
	ffmpegArgs := append([]string{}, inputArgs...)
	ffmpegArgs = append(ffmpegArgs, "-i", inputURL)
	ffmpegArgs = append(ffmpegArgs, outputArgs...)
	ffmpegArgs = append(ffmpegArgs, session.ManifestPath)

	// Start the session
	if err := session.StartLiveWithConfig(ffmpegArgs, inputURL, effectiveConfig); err != nil {
		return nil, fmt.Errorf("failed to start live stream session: %w", err)
	}

	// Store session
	m.sessions.Store(key, session)
	m.liveMu.Lock()
	if m.pausedSession == nil {
		// No paused session yet
	} else {
		// This should not happen for live streams (only one paused at a time)
	}
	m.liveMu.Unlock()

	// Start retention watchdog for DVR limits
	go session.monitorRetentionLimits()

	m.logger.Info("Created live stream session",
		"session_id", session.ID,
		"library_id", params.LibraryID,
		"channel_id", params.ChannelID)

	return session, nil
}

// GetSession retrieves an existing session by media ID, quality, and audio track.
func (m *Manager) GetSession(mediaID int64, quality string, audioTrackIndex int) (*TranscodeSession, error) {
	key := sessionKey(mediaID, quality, audioTrackIndex)

	if existing, ok := m.sessions.Load(key); ok {
		session := existing.(*TranscodeSession)
		session.UpdateLastAccessed()
		return session, nil
	}

	// If audio track not specified, also try to find a session with default audio
	if audioTrackIndex < 0 {
		defaultKey := sessionKey(mediaID, quality, 0)
		if existing, ok := m.sessions.Load(defaultKey); ok {
			session := existing.(*TranscodeSession)
			session.UpdateLastAccessed()
			return session, nil
		}
	}

	return nil, fmt.Errorf("no active transcode session for media %d quality %s", mediaID, quality)
}

// StopSession stops a specific session and removes it from the manager.
func (m *Manager) StopSession(mediaID int64, quality string, audioTrackIndex int) error {
	key := sessionKey(mediaID, quality, audioTrackIndex)

	if existing, ok := m.sessions.Load(key); ok {
		session := existing.(*TranscodeSession)
		session.Stop()
		m.sessions.Delete(key)

		if m.logStore != nil {
			m.logStore.CloseLogWriter(session.ID)
		}

		m.cleanupOutputDir(session.OutputDir, session.ID)

		m.logger.Info("Stopped transcode session", "session_id", session.ID)
		return nil
	}

	return fmt.Errorf("session not found")
}

// StopAllSessions stops all active sessions.
func (m *Manager) StopAllSessions() {
	m.logger.Info("Stopping all transcode sessions")

	m.sessions.Range(func(key, value any) bool {
		session := value.(*TranscodeSession)
		session.Stop()

		if m.logStore != nil {
			m.logStore.CloseLogWriter(session.ID)
		}

		m.cleanupOutputDir(session.OutputDir, session.ID)

		return true
	})

	m.sessions = sync.Map{}
}

// GetStats returns statistics about active sessions.
func (m *Manager) GetStats() map[string]interface{} {
	activeCount := 0
	m.sessions.Range(func(key, value interface{}) bool {
		activeCount++
		return true
	})

	return map[string]interface{}{
		"active_sessions": activeCount,
	}
}

// GetSessionOutputPath returns the path where a session's segments are stored.
func (m *Manager) GetSessionOutputPath(mediaID int64, quality string, audioTrackIndex int) (string, error) {
	key := sessionKey(mediaID, quality, audioTrackIndex)

	if existing, ok := m.sessions.Load(key); ok {
		session := existing.(*TranscodeSession)
		return session.OutputDir, nil
	}

	if audioTrackIndex < 0 {
		defaultKey := sessionKey(mediaID, quality, 0)
		if existing, ok := m.sessions.Load(defaultKey); ok {
			session := existing.(*TranscodeSession)
			return session.OutputDir, nil
		}
	}

	return "", fmt.Errorf("no active session for media %d quality %s", mediaID, quality)
}

// sessionKey generates a unique key for a media/quality/audio combination.
func sessionKey(mediaID int64, quality string, audioTrackIndex int) string {
	if audioTrackIndex > 0 {
		return fmt.Sprintf("%d:%s:audio%d", mediaID, quality, audioTrackIndex)
	}
	return fmt.Sprintf("%d:%s", mediaID, quality)
}

// convertProbeToHLSVideoInfo converts videoinfo.VideoInfo to hls.VideoInfo.
// Since videoinfo.VideoInfo embeds hls.VideoInfo, this is a zero-copy operation.
func convertProbeToHLSVideoInfo(v *videoinfo.VideoInfo) *hls.VideoInfo {
	if v == nil {
		return nil
	}
	return v.ToHLSVideoInfo()
}

// WarmupGPU pre-initializes GPU resources for HDR tone mapping.
// This eliminates the 1-3 second first-run delay for Vulkan/OpenCL initialization.
// Should be called once during server startup for NVENC/QSV with tone mapping enabled.
// Runs asynchronously and logs the result.
func (m *Manager) WarmupGPU() {
	// Only warm up if we have hardware acceleration and tone mapping enabled
	hwAccel := hls.HardwareAccel(m.hwAccel)
	if hwAccel != hls.AccelNVENC && hwAccel != hls.AccelQSV {
		m.logger.Debug("GPU warmup skipped - no NVENC/QSV hardware acceleration")
		return
	}

	if m.config == nil || !m.config.ToneMappingEnabled {
		m.logger.Debug("GPU warmup skipped - tone mapping not enabled")
		return
	}

	backend := m.config.ToneMappingBackend
	if backend == "" {
		backend = "auto"
	}

	// Run warmup asynchronously to not block server startup
	go func() {
		start := time.Now()
		m.logger.Info("Starting GPU warmup for HDR tone mapping",
			"backend", backend,
			"hw_accel", m.hwAccel)

		var args []string
		ffmpegPath := "ffmpeg"
		if m.config.FFmpegPaths != nil && m.config.FFmpegPaths.FFmpeg != "" {
			ffmpegPath = m.config.FFmpegPaths.FFmpeg
		}

		// Build a minimal FFmpeg command that initializes the GPU filter chain
		// Uses lavfi color source to avoid needing an input file
		switch backend {
		case "vulkan":
			// Initialize Vulkan device - this is what takes ~1.9s on first run
			args = []string{
				"-init_hw_device", "vulkan",
				"-f", "lavfi", "-i", "color=black:s=64x64:d=0.1",
				"-frames:v", "1",
				"-f", "null", "-",
			}
		case "opencl", "auto":
			// Initialize OpenCL device - this is what takes ~3s on first run
			args = []string{
				"-init_hw_device", "opencl",
				"-f", "lavfi", "-i", "color=black:s=64x64:d=0.1",
				"-frames:v", "1",
				"-f", "null", "-",
			}
		default:
			m.logger.Debug("GPU warmup skipped - backend doesn't need warmup",
				"backend", backend)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, ffmpegPath, args...)

		// Set library path if configured
		if m.config.FFmpegPaths != nil && m.config.FFmpegPaths.LibPath != "" {
			cmd.Env = append(cmd.Environ(), "LD_LIBRARY_PATH="+m.config.FFmpegPaths.LibPath)
		}

		err := cmd.Run()
		elapsed := time.Since(start)

		if err != nil {
			m.logger.Warn("GPU warmup failed",
				"backend", backend,
				"error", err,
				"elapsed_ms", elapsed.Milliseconds(),
				"note", "First HDR transcode may have slower startup")
		} else {
			m.logger.Info("GPU warmup completed",
				"backend", backend,
				"elapsed_ms", elapsed.Milliseconds())
		}
	}()
}

// LiveStreamStatus represents the status of a live stream session.
type LiveStreamStatus struct {
	State            LiveStreamState
	Position         float64
	RetainedSegments int
}

// liveStreamKey generates a unique key for a live stream session.
func liveStreamKey(libraryID, channelID int64) string {
	return fmt.Sprintf("live:%d:%d", libraryID, channelID)
}

// GetLiveStreamSession retrieves an active live stream session.
func (m *Manager) GetLiveStreamSession(libraryID, channelID int64) (*TranscodeSession, error) {
	key := liveStreamKey(libraryID, channelID)

	if existing, ok := m.sessions.Load(key); ok {
		session := existing.(*TranscodeSession)
		session.UpdateLastAccessed()
		return session, nil
	}

	return nil, fmt.Errorf("no active live stream for library %d channel %d", libraryID, channelID)
}

// GetLiveStreamOutputPath returns the output directory for a live stream session.
func (m *Manager) GetLiveStreamOutputPath(libraryID, channelID int64) (string, error) {
	session, err := m.GetLiveStreamSession(libraryID, channelID)
	if err != nil {
		return "", err
	}
	return session.OutputDir, nil
}

// GetLiveStreamStatus returns the status of a live stream session.
func (m *Manager) GetLiveStreamStatus(libraryID, channelID int64) (*LiveStreamStatus, error) {
	session, err := m.GetLiveStreamSession(libraryID, channelID)
	if err != nil {
		return nil, err
	}

	return &LiveStreamStatus{
		State:            session.GetLiveState(),
		Position:         session.GetPausedPosition(),
		RetainedSegments: session.GetRetainedSegments(),
	}, nil
}

// CheckPauseConflict checks if another session is already paused.
func (m *Manager) CheckPauseConflict(currentSession *TranscodeSession) error {
	m.liveMu.Lock()
	defer m.liveMu.Unlock()

	if m.pausedSession != nil && m.pausedSession != currentSession {
		return fmt.Errorf("another channel is currently paused")
	}
	return nil
}

// RegisterPausedSession registers a session as the currently paused session.
func (m *Manager) RegisterPausedSession(sess *TranscodeSession) {
	m.liveMu.Lock()
	defer m.liveMu.Unlock()
	m.pausedSession = sess
}

// ClearPausedSession clears the paused session if it matches the given session.
func (m *Manager) ClearPausedSession(sess *TranscodeSession) {
	m.liveMu.Lock()
	defer m.liveMu.Unlock()
	if m.pausedSession == sess {
		m.pausedSession = nil
	}
}

// StopLiveStreamSession stops a live stream session.
func (m *Manager) StopLiveStreamSession(libraryID, channelID int64) error {
	key := liveStreamKey(libraryID, channelID)

	if existing, ok := m.sessions.Load(key); ok {
		session := existing.(*TranscodeSession)

		// Clear paused session if this is the one
		m.liveMu.Lock()
		if m.pausedSession == session {
			m.pausedSession = nil
		}
		m.liveMu.Unlock()

		session.StopLive()

		// Clean up output directory
		m.cleanupOutputDir(session.OutputDir, session.ID)

		// Remove from sessions map
		m.sessions.Delete(key)

		if m.logStore != nil {
			m.logStore.CloseLogWriter(session.ID)
		}

		m.logger.Info("Stopped live stream session",
			"session_id", session.ID,
			"library_id", libraryID,
			"channel_id", channelID)
		return nil
	}

	return fmt.Errorf("no active live stream for library %d channel %d", libraryID, channelID)
}
