package livetv

import (
	"strings"
	"testing"
)

func TestParseM3UChannel(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  M3UAttribute
	}{
		{
			name:  "full attributes with tvg-id and logo",
			input: `#EXTINF:-1 tvg-id="yle1" tvg-logo="http://logo.tv/yle1.png" group-title="Finnish",Yle TV1`,
			want:  M3UAttribute{Name: "Yle TV1"},
		},
		{
			name:  "with channel number",
			input: `#EXTINF:-1 tvg-chno="5" tvg-id="mtv3" group-title="Entertainment",MTV3`,
			want:  M3UAttribute{Name: "MTV3"},
		},
		{
			name:  "simple format no attributes",
			input: `#EXTINF:0,Yle TV1 HD`,
			want:  M3UAttribute{Name: "Yle TV1 HD"},
		},
		{
			name:  "no comma separator returns raw trailing as name",
			input: `#EXTINF:-1 Yle TV1`,
			want:  M3UAttribute{Name: "Yle TV1"},
		},
		{
			name:  "not an EXTINF line returns empty",
			input: `#EXTM3U`,
			want:  M3UAttribute{},
		},
		{
			name:  "empty line",
			input: ``,
			want:  M3UAttribute{},
		},
		{
			name:  "tvg-chno with non-numeric value does not panic",
			input: `#EXTINF:-1 tvg-chno="abc" group-title="News",News Channel`,
			want:  M3UAttribute{Name: "News Channel"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseM3UChannel(tt.input)
			if got != tt.want {
				t.Errorf("ParseM3UChannel() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseM3U(t *testing.T) {
	t.Run("standard playlist with multiple channels", func(t *testing.T) {
		input := `#EXTM3U
#EXTINF:-1 tvg-id="yle1" tvg-logo="http://logo.tv/yle1.png" group-title="Finnish",Yle TV1
http://stream.example.com/yle1.m3u8
#EXTINF:-1 tvg-chno="2" tvg-id="mtv3" group-title="Entertainment",MTV3
http://stream.example.com/mtv3.m3u8
#EXTINF:0,Radio Channel
http://stream.example.com/radio.mp3`
		got, err := ParseM3U(strings.NewReader(input))
		if err != nil {
			t.Fatalf("ParseM3U() error = %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("expected 3 entries, got %d", len(got))
		}
		if got[0].Name != "Yle TV1" || got[0].StreamURL != "http://stream.example.com/yle1.m3u8" {
			t.Errorf("entry 0 = %+v", got[0])
		}
		if got[1].Name != "MTV3" || got[1].StreamURL != "http://stream.example.com/mtv3.m3u8" {
			t.Errorf("entry 1 = %+v", got[1])
		}
	})

	t.Run("skips extvlcopt and hash comments", func(t *testing.T) {
		input := `#EXTM3U
#EXTVLCOPT:http-user-agent=Test
# some comment
#EXTINF:-1,Test Channel
http://stream.example.com/test`
		got, err := ParseM3U(strings.NewReader(input))
		if err != nil {
			t.Fatalf("ParseM3U() error = %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(got))
		}
	})

	t.Run("empty playlist returns no entries", func(t *testing.T) {
		got, err := ParseM3U(strings.NewReader("#EXTM3U\n"))
		if err != nil {
			t.Fatalf("ParseM3U() error = %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected 0 entries, got %d", len(got))
		}
	})

	t.Run("URL line without preceding EXTINF has empty name", func(t *testing.T) {
		input := `#EXTM3U
http://stream.example.com/unknown.m3u8`
		got, err := ParseM3U(strings.NewReader(input))
		if err != nil {
			t.Fatalf("ParseM3U() error = %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(got))
		}
		if got[0].Name != "" {
			t.Errorf("name = %q, want empty", got[0].Name)
		}
	})
}

func TestParseSatipStreamURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want SatipStreamParams
	}{
		{
			name: "full SAT>IP RTSP URL with freq and pids",
			url:  "rtsp://192.168.1.100/?src=1&freq=12000&pids=0,400,401",
			want: SatipStreamParams{Frequency: 12000, PIDs: "0,400,401"},
		},
		{
			name: "URL with freq only",
			url:  "rtsp://192.168.1.100/?src=1&freq=12000",
			want: SatipStreamParams{Frequency: 12000},
		},
		{
			name: "URL with pids only",
			url:  "rtsp://192.168.1.100/?src=1&pids=0,400",
			want: SatipStreamParams{PIDs: "0,400"},
		},
		{
			name: "non-SAT>IP URL",
			url:  "http://stream.example.com/test.m3u8",
			want: SatipStreamParams{},
		},
		{
			name: "URL with multiple params including freq and deep pids",
			url:  "rtsp://192.168.1.100:554/?freq=12100&msys=dvbs&pids=0,100,101,102",
			want: SatipStreamParams{Frequency: 12100, PIDs: "0,100,101,102"},
		},
		{
			name: "empty URL",
			url:  "",
			want: SatipStreamParams{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseSatipStreamURL(tt.url)
			if got != tt.want {
				t.Errorf("ParseSatipStreamURL() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestCorrectPIDs(t *testing.T) {
	tests := []struct {
		name        string
		streamURL   string
		correctPIDs string
		want        string
	}{
		{
			name:        "replaces pids parameter",
			streamURL:   "rtsp://192.168.1.100/?src=1&freq=12000&pids=0,400,401",
			correctPIDs: "0,100,101",
			want:        "rtsp://192.168.1.100/?src=1&freq=12000&pids=0,100,101",
		},
		{
			name:        "no pids parameter returns original",
			streamURL:   "rtsp://192.168.1.100/?src=1&freq=12000",
			correctPIDs: "0,100",
			want:        "rtsp://192.168.1.100/?src=1&freq=12000",
		},
		{
			name:        "pids at start of query params",
			streamURL:   "rtsp://192.168.1.100/?pids=0,1&freq=12000",
			correctPIDs: "0,2,3",
			want:        "rtsp://192.168.1.100/?pids=0,2,3&freq=12000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CorrectPIDs(tt.streamURL, tt.correctPIDs)
			if got != tt.want {
				t.Errorf("CorrectPIDs() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsChannelListURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"http://192.168.1.100/dvb/m3u/channellist.m3u8", true},
		{"http://192.168.1.100/dvb/m3u/channellist.m3u", true},
		{"http://192.168.1.100/dvb/m3u/list.m3u8", true},
		{"http://stream.example.com/yle1.m3u8", false},
		{"http://192.168.1.100/channellist.m3u", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := IsChannelListURL(tt.url)
			if got != tt.want {
				t.Errorf("IsChannelListURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestFilterChannelListEntry(t *testing.T) {
	t.Run("filters channellist entry and returns its URL", func(t *testing.T) {
		entries := []M3UEntry{
			{Name: "Channel 1", StreamURL: "http://stream.example.com/ch1.m3u8"},
			{Name: "Channel List", StreamURL: "http://192.168.1.100/dvb/m3u/list.m3u8"},
			{Name: "Channel 2", StreamURL: "http://stream.example.com/ch2.m3u8"},
		}
		filtered, channellistURL, err := FilterChannelListEntry(entries)
		if err != nil {
			t.Fatalf("FilterChannelListEntry() error = %v", err)
		}
		if len(filtered) != 2 {
			t.Fatalf("expected 2 filtered entries, got %d", len(filtered))
		}
		if channellistURL != "http://192.168.1.100/dvb/m3u/list.m3u8" {
			t.Errorf("channellistURL = %q", channellistURL)
		}
	})

	t.Run("no channellist entry returns all entries", func(t *testing.T) {
		entries := []M3UEntry{
			{Name: "Channel 1", StreamURL: "http://stream.example.com/ch1.m3u8"},
		}
		filtered, channellistURL, err := FilterChannelListEntry(entries)
		if err != nil {
			t.Fatalf("FilterChannelListEntry() error = %v", err)
		}
		if len(filtered) != 1 {
			t.Errorf("expected 1 filtered entry, got %d", len(filtered))
		}
		if channellistURL != "" {
			t.Errorf("channellistURL = %q, want empty", channellistURL)
		}
	})

	t.Run("only channellist entry returns empty filtered", func(t *testing.T) {
		entries := []M3UEntry{
			{StreamURL: "http://192.168.1.100/dvb/m3u/list.m3u8"},
		}
		filtered, channellistURL, err := FilterChannelListEntry(entries)
		if err != nil {
			t.Fatalf("FilterChannelListEntry() error = %v", err)
		}
		if len(filtered) != 0 {
			t.Errorf("expected 0 filtered entries, got %d", len(filtered))
		}
		if channellistURL == "" {
			t.Error("channellistURL should not be empty")
		}
	})
}

func TestToDomainChannel(t *testing.T) {
	entry := M3UEntry{
		ChannelNumber: 5,
		Name:          "Test Channel",
		StreamURL:     "http://stream.example.com/test.m3u8",
		LogoURL:       "http://logo.tv/test.png",
		Group:         "News",
		EPGChannelID:  "test_channel",
	}
	ch := entry.ToDomainChannel(42)
	if ch.LibraryID != 42 {
		t.Errorf("LibraryID = %d, want 42", ch.LibraryID)
	}
	if ch.ChannelNumber != 5 {
		t.Errorf("ChannelNumber = %d, want 5", ch.ChannelNumber)
	}
	if ch.Name != "Test Channel" {
		t.Errorf("Name = %q", ch.Name)
	}
	if ch.StreamURL != "http://stream.example.com/test.m3u8" {
		t.Errorf("StreamURL = %q", ch.StreamURL)
	}
	if ch.LogoURL != "http://logo.tv/test.png" {
		t.Errorf("LogoURL = %q", ch.LogoURL)
	}
	if ch.Group != "News" {
		t.Errorf("Group = %q", ch.Group)
	}
	if ch.EPGChannelID != "test_channel" {
		t.Errorf("EPGChannelID = %q", ch.EPGChannelID)
	}
	if !ch.Enabled {
		t.Error("Enabled should be true")
	}
}

func TestFormatPIDsForDisplay(t *testing.T) {
	tests := []struct {
		pids string
		want string
	}{
		{"0,400,401", "0x0000, 0x0190, 0x0191"},
		{"0", "0x0000"},
		{"", ""},
		{"abc", "abc"},
		{"0,abc,401", "0x0000, abc, 0x0191"},
	}
	for _, tt := range tests {
		t.Run(tt.pids, func(t *testing.T) {
			got := FormatPIDsForDisplay(tt.pids)
			if got != tt.want {
				t.Errorf("FormatPIDsForDisplay(%q) = %q, want %q", tt.pids, got, tt.want)
			}
		})
	}
}
