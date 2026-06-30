package strategy

import (
	"fmt"

	"github.com/mantonx/viewra/internal/infrastructure/transcoding/profile"
)

// LiveStreamStrategy provides the streaming strategy for live RTSP streams.
// Live streams are always transcoded to HLS with a single "live" quality profile.
// Supports DVR with segment retention for pause/resume functionality.
type LiveStreamStrategy struct {
	Profile *profile.AdaptiveProfile
}

// NewLiveStreamStrategy creates a new live stream strategy with default settings.
// The profile is configured for 1080p H.264 transcoding with 2-second segments for DVR.
func NewLiveStreamStrategy() *LiveStreamStrategy {
	return &LiveStreamStrategy{
		Profile: &profile.AdaptiveProfile{
			ID:              "live",
			DisplayName:     "Live",
			Width:           1920,
			Height:          1080,
			VideoBitrate:    5_000_000,   // 5 Mbps
			VideoMaxRate:    5_500_000,   // 110%
			VideoBufSize:    10_000_000,  // 2x
			EnableHWAccel:   true,
			EnableFastStart: true,
			SegmentDuration: 2,          // 2-second segments for DVR
			GOPSize:         48,         // 2 seconds at 24fps
			FrameRate:       24.0,
			AspectRatio:     "16:9",
			AudioBitrate:    128_000,
			AudioChannels:   2,
			AudioSampleRate: 48000,
			AudioCodec:      "aac",
			MaxAudioChannels: 2,
			PreserveMultiCh: false,
			PreferredCodec:  "h264",
			FallbackCodecs:  []string{},
			Preset:          "veryfast",
			CRF:             23,
			MinNetworkMbps:  5.0,
			MinScreenWidth:  1920,
			MinScreenHeight: 1080,
			RecommendedFor:  []string{"desktop", "tv"},
			Description:     "Live TV stream with DVR support",
			QualityTier:     "high",
			DataUsageMBPerHour: (5_000_000 + 128_000) / 8 * 3600 / 1_000_000,
		},
	}
}

// GetStrategy always returns Transcode for live RTSP streams.
func (s *LiveStreamStrategy) GetStrategy() StreamStrategy {
	return Transcode
}

// GetProfile returns the adaptive profile for live streaming.
func (s *LiveStreamStrategy) GetProfile() *profile.AdaptiveProfile {
	return s.Profile
}

// GetReason returns the reason for the transcode decision.
func (s *LiveStreamStrategy) GetReason() string {
	return "Live RTSP stream requires transcoding to HLS with DVR support"
}

// GetSegmentDuration returns the segment duration in seconds.
// 2-second segments provide a good balance between latency and DVR window size.
func (s *LiveStreamStrategy) GetSegmentDuration() int {
	return s.Profile.SegmentDuration
}

// GetFFmpegInputArgs returns RTSP-specific FFmpeg input arguments.
func (s *LiveStreamStrategy) GetFFmpegInputArgs() []string {
	return []string{
		"-rtsp_transport", "tcp",
		"-stimeout", "5000000",       // 5 second timeout
		"-buffer_size", "1024000",     // 1MB buffer
		"-re",                         // Read at native frame rate
		"-fflags", "+genpts",
		"-avoid_negative_ts", "make_zero",
		"-use_wallclock_as_timestamps", "1", // For PROGRAM-DATE-TIME
	}
}

// GetFFmpegOutputArgs returns HLS output arguments optimized for live DVR.
func (s *LiveStreamStrategy) GetFFmpegOutputArgs(segmentDuration int, outputDir string) []string {
	return []string{
		"-c:v", "libx264",
		"-preset", s.Profile.Preset,
		"-crf", fmt.Sprintf("%d", s.Profile.CRF),
		"-b:v", fmt.Sprintf("%d", s.Profile.VideoBitrate),
		"-maxrate", fmt.Sprintf("%d", s.Profile.VideoMaxRate),
		"-bufsize", fmt.Sprintf("%d", s.Profile.VideoBufSize),
		"-g", fmt.Sprintf("%d", s.Profile.GOPSize),
		"-keyint_min", fmt.Sprintf("%d", s.Profile.GOPSize/2),
		"-sc_threshold", "0",
		"-r", fmt.Sprintf("%.0f", s.Profile.FrameRate),
		"-c:a", "aac",
		"-b:a", fmt.Sprintf("%d", s.Profile.AudioBitrate),
		"-ac", fmt.Sprintf("%d", s.Profile.AudioChannels),
		"-ar", fmt.Sprintf("%d", s.Profile.AudioSampleRate),
		"-f", "hls",
		"-hls_time", fmt.Sprintf("%d", segmentDuration),
		"-hls_list_size", "1800",            // Keep up to 1800 segments (60 min at 2s)
		"-hls_flags", "program_date_time+append_list+delete_segments",
		"-hls_segment_filename", outputDir + "/seg_%06d.ts",
		"-hls_playlist_type", "event",
		"-use_wallclock_as_timestamps", "1",
		"-strftime_mkdir", "1",
	}
}

// LiveStreamStrategyInstance provides a singleton instance for convenience.
var LiveStreamStrategyInstance = NewLiveStreamStrategy()

// GetLiveStreamStrategy returns the live stream strategy instance.
func GetLiveStreamStrategy() *LiveStreamStrategy {
	return LiveStreamStrategyInstance
}