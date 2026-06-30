package livetv

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mantonx/viewra/internal/domain/library"
	"github.com/mantonx/viewra/internal/domain/livetv"
)

func TestCorrectChannelPIDsFromList(t *testing.T) {
	t.Run("corrects PIDs when frequency matches", func(t *testing.T) {
		ch := &livetv.Channel{
			StreamURL: "rtsp://192.168.1.100/?src=1&freq=12000&pids=0,400",
		}
		freqToPIDs := map[int]string{12000: "0,100,101"}
		changed := CorrectChannelPIDsFromList(ch, freqToPIDs)
		if !changed {
			t.Error("expected channel to be corrected")
		}
		if ch.StreamURL != "rtsp://192.168.1.100/?src=1&freq=12000&pids=0,100,101" {
			t.Errorf("StreamURL = %q", ch.StreamURL)
		}
	})

	t.Run("no change when PIDs already match", func(t *testing.T) {
		ch := &livetv.Channel{
			StreamURL: "rtsp://192.168.1.100/?freq=12000&pids=0,400",
		}
		freqToPIDs := map[int]string{12000: "0,400"}
		changed := CorrectChannelPIDsFromList(ch, freqToPIDs)
		if changed {
			t.Error("expected no change when PIDs already match")
		}
	})

	t.Run("no change when frequency not in map", func(t *testing.T) {
		ch := &livetv.Channel{
			StreamURL: "rtsp://192.168.1.100/?freq=13000&pids=0,400",
		}
		freqToPIDs := map[int]string{12000: "0,100"}
		changed := CorrectChannelPIDsFromList(ch, freqToPIDs)
		if changed {
			t.Error("expected no change for unknown frequency")
		}
	})

	t.Run("no change when URL has no freq parameter", func(t *testing.T) {
		ch := &livetv.Channel{
			StreamURL: "http://stream.example.com/channel.m3u8",
		}
		changed := CorrectChannelPIDsFromList(ch, map[int]string{12000: "0,100"})
		if changed {
			t.Error("expected no change for non-SAT>IP URL")
		}
	})
}

func TestFormatPIDsForDisplay_Satip(t *testing.T) {
	// Verify FormatPIDsForDisplay is accessible (shared with satip_pid.go)
	got := FormatPIDsForDisplay("0,400")
	if got != "0x0000, 0x0190" {
		t.Errorf("got %q", got)
	}
}

type mockLibraryRepoForSatip struct {
	library.Repository
	updatedID      int64
	updatedEnabled bool
	updatedConfig  *library.MonitoringConfig
}

func (m *mockLibraryRepoForSatip) UpdateMonitoring(ctx context.Context, id int64, enabled bool, config *library.MonitoringConfig) error {
	m.updatedID = id
	m.updatedEnabled = enabled
	m.updatedConfig = config
	return nil
}

type mockChannelRepoForSatip struct {
	livetv.ChannelRepository
	upserted []*livetv.Channel
}

func (m *mockChannelRepoForSatip) Upsert(ctx context.Context, ch *livetv.Channel) error {
	m.upserted = append(m.upserted, ch)
	return nil
}

func TestCorrectSATIPPIDs(t *testing.T) {
	t.Run("no channellist URL returns entries unchanged", func(t *testing.T) {
		entries := []M3UEntry{
			{Name: "Ch1", StreamURL: "http://stream.example.com/ch1.m3u8"},
		}
		lib := &library.Library{ID: 1, MonitoringConfig: &library.MonitoringConfig{}}
		result, err := CorrectSATIPPIDs(context.Background(), entries, lib, nil, nil, nil)
		if err != nil {
			t.Fatalf("CorrectSATIPPIDs() error = %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(result))
		}
	})

	t.Run("detects channellist in entries when lib has none", func(t *testing.T) {
		entries := []M3UEntry{
			{Name: "Ch1", StreamURL: "http://stream.example.com/ch1.m3u8"},
		}
		lib := &library.Library{ID: 1}
		result, err := CorrectSATIPPIDs(context.Background(), entries, lib, nil, nil, nil)
		if err != nil {
			t.Fatalf("CorrectSATIPPIDs() error = %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(result))
		}
	})

	t.Run("channellist in lib config when entries have none", func(t *testing.T) {
		entries := []M3UEntry{
			{Name: "Ch1", StreamURL: "http://stream.example.com/ch1.m3u8"},
		}
		lib := &library.Library{
			ID: 1,
			MonitoringConfig: &library.MonitoringConfig{
				SatipChannelListURL: "http://127.0.0.1:1/dvb/m3u/list.m3u8",
			},
		}
		libRepo := &mockLibraryRepoForSatip{}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		result, err := CorrectSATIPPIDs(ctx, entries, lib, libRepo, nil, &http.Client{})
		if err == nil {
			t.Fatal("expected error from channellist fetch failure")
		}
		_ = result
	})

	t.Run("filters channellist entry and persists URL", func(t *testing.T) {
		var channellistURL string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/dvb/m3u/list.m3u8" {
				w.Write([]byte(`#EXTM3U
#EXTINF:-1,Ch1
rtsp://192.168.1.100/?freq=12000&pids=0,400`))
			}
		}))
		defer server.Close()
		channellistURL = server.URL + "/dvb/m3u/list.m3u8"

		entries := []M3UEntry{
			{Name: "Ch1", StreamURL: "http://stream.example.com/ch1.m3u8"},
			{StreamURL: channellistURL},
		}
		lib := &library.Library{ID: 1}
		libRepo := &mockLibraryRepoForSatip{}
		result, err := CorrectSATIPPIDs(context.Background(), entries, lib, libRepo, nil, server.Client())
		if err != nil {
			t.Fatalf("CorrectSATIPPIDs() error = %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 entry after filtering, got %d", len(result))
		}
		if libRepo.updatedConfig == nil || libRepo.updatedConfig.SatipChannelListURL == "" {
			t.Error("expected SatipChannelListURL to be persisted")
		}
	})
}

func TestFetchSatipChannelList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`#EXTM3U
#EXTINF:-1,Channel 1
rtsp://192.168.1.100/?freq=12000&pids=0,400
#EXTINF:-1,Channel 2
rtsp://192.168.1.100/?freq=12100&pids=0,401,402`))
	}))
	defer server.Close()

	freqToPIDs, err := FetchSatipChannelList(context.Background(), server.URL, server.Client())
	if err != nil {
		t.Fatalf("FetchSatipChannelList() error = %v", err)
	}
	if len(freqToPIDs) != 2 {
		t.Fatalf("expected 2 frequencies, got %d", len(freqToPIDs))
	}
	if freqToPIDs[12000] != "0,400" {
		t.Errorf("freq 12000 PIDs = %q", freqToPIDs[12000])
	}
	if freqToPIDs[12100] != "0,401,402" {
		t.Errorf("freq 12100 PIDs = %q", freqToPIDs[12100])
	}
}

func TestFetchSatipChannelList_PrefersMorePIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`#EXTM3U
#EXTINF:-1,A
rtsp://192.168.1.100/?freq=12000&pids=0,400
#EXTINF:-1,B
rtsp://192.168.1.100/?freq=12000&pids=0,400,401,402`))
	}))
	defer server.Close()

	freqToPIDs, err := FetchSatipChannelList(context.Background(), server.URL, server.Client())
	if err != nil {
		t.Fatalf("FetchSatipChannelList() error = %v", err)
	}
	if freqToPIDs[12000] != "0,400,401,402" {
		t.Errorf("expected longer PID list, got %q", freqToPIDs[12000])
	}
}

func TestFetchSatipChannelList_InvalidURL(t *testing.T) {
	_, err := FetchSatipChannelList(context.Background(), "not-a-url", &http.Client{})
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestFetchSatipChannelList_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := FetchSatipChannelList(context.Background(), server.URL, server.Client())
	if err == nil {
		t.Error("expected error when server returns error")
	}
}
