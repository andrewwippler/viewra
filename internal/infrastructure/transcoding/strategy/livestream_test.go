package strategy

import (
	"testing"
)

func TestLiveStreamStrategyProfile(t *testing.T) {
	s := NewLiveStreamStrategy()
	p := s.GetProfile()
	if p == nil {
		t.Fatal("GetProfile() returned nil")
	}
	if p.ID != "live" {
		t.Errorf("Profile.ID = %q, want %q", p.ID, "live")
	}
	if p.Width != 1920 || p.Height != 1080 {
		t.Errorf("resolution = %dx%d, want 1920x1080", p.Width, p.Height)
	}
	if p.VideoBitrate != 5_000_000 {
		t.Errorf("VideoBitrate = %d, want 5000000", p.VideoBitrate)
	}
	if p.SegmentDuration != 2 {
		t.Errorf("SegmentDuration = %d, want 2", p.SegmentDuration)
	}
	if p.AudioCodec != "aac" {
		t.Errorf("AudioCodec = %q, want %q", p.AudioCodec, "aac")
	}
	if p.PreferredCodec != "h264" {
		t.Errorf("PreferredCodec = %q, want %q", p.PreferredCodec, "h264")
	}
	if p.CRF != 23 {
		t.Errorf("CRF = %d, want 23", p.CRF)
	}
	if p.Preset != "veryfast" {
		t.Errorf("Preset = %q, want %q", p.Preset, "veryfast")
	}
	if p.GOPSize != 48 {
		t.Errorf("GOPSize = %d, want 48", p.GOPSize)
	}
	if p.FrameRate != 24.0 {
		t.Errorf("FrameRate = %f, want 24.0", p.FrameRate)
	}
}

func TestLiveStreamStrategyGetStrategy(t *testing.T) {
	s := NewLiveStreamStrategy()
	if got := s.GetStrategy(); got != Transcode {
		t.Errorf("GetStrategy() = %v, want %v", got, Transcode)
	}
}

func TestLiveStreamStrategyGetSegmentDuration(t *testing.T) {
	s := NewLiveStreamStrategy()
	if got := s.GetSegmentDuration(); got != 2 {
		t.Errorf("GetSegmentDuration() = %d, want 2", got)
	}
}

func TestLiveStreamStrategyGetReason(t *testing.T) {
	s := NewLiveStreamStrategy()
	reason := s.GetReason()
	if reason == "" {
		t.Error("GetReason() returned empty")
	}
}

func TestLiveStreamStrategyFFmpegInputArgs(t *testing.T) {
	s := NewLiveStreamStrategy()
	args := s.GetFFmpegInputArgs()
	if len(args) == 0 {
		t.Fatal("GetFFmpegInputArgs() returned empty")
	}

	requiredFlags := []string{"-rtsp_transport", "tcp", "-re", "-fflags", "+genpts"}
	for _, flag := range requiredFlags {
		found := false
		for _, arg := range args {
			if arg == flag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing required input flag: %s", flag)
		}
	}
}

func TestLiveStreamStrategyFFmpegOutputArgs(t *testing.T) {
	s := NewLiveStreamStrategy()
	args := s.GetFFmpegOutputArgs(2, "/tmp/live")
	if len(args) == 0 {
		t.Fatal("GetFFmpegOutputArgs() returned empty")
	}

	requiredFlags := []string{"-c:v", "libx264", "-c:a", "aac", "-f", "hls", "-preset", "veryfast"}
	for _, flag := range requiredFlags {
		found := false
		for _, arg := range args {
			if arg == flag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing required output flag: %s", flag)
		}
	}

	// Check that segment duration is used
	expectedHlsTime := "-hls_time"
	found := false
	for _, arg := range args {
		if arg == expectedHlsTime {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("missing -hls_time flag")
	}

	// Check output dir is used in segment filename
	hasSegmentFilename := false
	for _, arg := range args {
		if arg == "/tmp/live/seg_%06d.ts" {
			hasSegmentFilename = true
			break
		}
	}
	if !hasSegmentFilename {
		t.Error("expected segment filename with output dir")
	}
}

func TestLiveStreamStrategySingleton(t *testing.T) {
	s1 := GetLiveStreamStrategy()
	s2 := GetLiveStreamStrategy()
	if s1 != s2 {
		t.Error("GetLiveStreamStrategy() should return singleton")
	}
	if s1 != LiveStreamStrategyInstance {
		t.Error("GetLiveStreamStrategy() should return LiveStreamStrategyInstance")
	}
}
