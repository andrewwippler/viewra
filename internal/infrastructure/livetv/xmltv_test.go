package livetv

import (
	"strings"
	"testing"
	"time"
)

func TestParseXMLTVTime(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{
			name:  "UTC time",
			input: "20260613040200 +0000",
			want:  time.Date(2026, 6, 13, 4, 2, 0, 0, time.UTC),
		},
		{
			name:  "positive offset",
			input: "20260613040200 +0300",
			want:  time.Date(2026, 6, 13, 4, 2, 0, 0, time.FixedZone("+0300", 3*3600)),
		},
		{
			name:  "negative offset",
			input: "20260613040200 -0500",
			want:  time.Date(2026, 6, 13, 4, 2, 0, 0, time.FixedZone("-0500", -5*3600)),
		},
		{
			name:  "without timezone defaults to UTC",
			input: "20260613040200",
			want:  time.Date(2026, 6, 13, 4, 2, 0, 0, time.UTC),
		},
		{
			name:  "short timestamp returns zero",
			input: "20260613",
			want:  time.Time{},
		},
		{
			name:  "empty string returns zero",
			input: "",
			want:  time.Time{},
		},
		{
			name:  "invalid month normalizes",
			input: "20261301000000 +0000",
			want:  time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseXMLTVTime(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseXMLTVTime() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !got.Equal(tt.want) {
				t.Errorf("parseXMLTVTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseXMLTV(t *testing.T) {
	t.Run("parses valid XMLTV with channels and programmes", func(t *testing.T) {
		input := `<?xml version="1.0" encoding="utf-8"?>
<tv>
  <channel id="yle1.example.com">
    <display-name lang="fi">Yle TV1</display-name>
  </channel>
  <channel id="mtv3.example.com">
    <display-name lang="fi">MTV3</display-name>
  </channel>
  <programme start="20260613040200 +0300" stop="20260613050000 +0300" channel="yle1.example.com">
    <title>Morning News</title>
    <sub-title>Weekend Edition</sub-title>
    <desc>Latest headlines and weather</desc>
    <category>News</category>
    <episode-num system="xmltv_ns">1.0.0</episode-num>
  </programme>
  <programme start="20260613050000 +0300" stop="20260613060000 +0300" channel="mtv3.example.com">
    <title>Movie Night</title>
    <category>Movie</category>
    <desc>A great film</desc>
  </programme>
</tv>`
		result, err := ParseXMLTV(strings.NewReader(input))
		if err != nil {
			t.Fatalf("ParseXMLTV() error = %v", err)
		}
		if len(result.ChannelMap) != 2 {
			t.Errorf("expected 2 channels, got %d", len(result.ChannelMap))
		}
		if result.ChannelMap["yle1.example.com"] != "Yle TV1" {
			t.Errorf("channel name = %q", result.ChannelMap["yle1.example.com"])
		}
		if len(result.Programs) != 2 {
			t.Fatalf("expected 2 programs, got %d", len(result.Programs))
		}
		if result.Programs[0].Title != "Morning News" {
			t.Errorf("program title = %q", result.Programs[0].Title)
		}
		if result.Programs[0].SubTitle != "Weekend Edition" {
			t.Errorf("program subtitle = %q", result.Programs[0].SubTitle)
		}
	})

	t.Run("channel without display-name uses ID as name", func(t *testing.T) {
		input := `<?xml version="1.0"?>
<tv>
  <channel id="channel1">
  </channel>
</tv>`
		result, err := ParseXMLTV(strings.NewReader(input))
		if err != nil {
			t.Fatalf("ParseXMLTV() error = %v", err)
		}
		if result.ChannelMap["channel1"] != "channel1" {
			t.Errorf("expected channel1, got %q", result.ChannelMap["channel1"])
		}
	})

	t.Run("empty TV returns empty result", func(t *testing.T) {
		input := `<?xml version="1.0"?><tv></tv>`
		result, err := ParseXMLTV(strings.NewReader(input))
		if err != nil {
			t.Fatalf("ParseXMLTV() error = %v", err)
		}
		if len(result.Programs) != 0 {
			t.Errorf("expected 0 programs, got %d", len(result.Programs))
		}
	})

	t.Run("invalid XML returns error", func(t *testing.T) {
		_, err := ParseXMLTV(strings.NewReader("not xml"))
		if err == nil {
			t.Error("expected error for invalid XML")
		}
	})

	t.Run("invalid start time returns zero time but is not skipped", func(t *testing.T) {
		input := `<?xml version="1.0"?>
<tv>
  <programme start="invalid" stop="20260613050000 +0300" channel="ch1">
    <title>Bad Start</title>
  </programme>
  <programme start="20260613040200 +0300" stop="20260613050000 +0300" channel="ch1">
    <title>Good Program</title>
  </programme>
</tv>`
		result, err := ParseXMLTV(strings.NewReader(input))
		if err != nil {
			t.Fatalf("ParseXMLTV() error = %v", err)
		}
		if len(result.Programs) != 2 {
			t.Fatalf("expected 2 programs, got %d", len(result.Programs))
		}
		// Both programs are parsed; the Bad Start has zero start time but no error
		if result.Programs[1].Title != "Good Program" {
			t.Errorf("title = %q", result.Programs[1].Title)
		}
	})
}

func TestToDomainPrograms(t *testing.T) {
	now := time.Now()
	programs := []XMLTVProgram{
		{
			ChannelID: "ch1",
			StartTime: now,
			EndTime:   now.Add(1 * time.Hour),
			Title:     "News",
			Category:  "News",
		},
		{
			ChannelID: "ch2",
			StartTime: now,
			EndTime:   now.Add(2 * time.Hour),
			Title:     "Movie",
			Category:  "Movie",
		},
		{
			ChannelID: "ch3", // no mapping
			StartTime: now,
			EndTime:   now.Add(1 * time.Hour),
			Title:     "Unmapped",
		},
	}

	epgToChannelID := map[string]int64{
		"ch1": 100,
		"ch2": 200,
	}

	domainPrograms := ToDomainPrograms(programs, epgToChannelID)
	if len(domainPrograms) != 2 {
		t.Fatalf("expected 2 domain programs, got %d", len(domainPrograms))
	}
	if domainPrograms[0].ChannelID != 100 || domainPrograms[0].Title != "News" {
		t.Errorf("program 0 = %+v", domainPrograms[0])
	}
	if domainPrograms[1].ChannelID != 200 || domainPrograms[1].Title != "Movie" {
		t.Errorf("program 1 = %+v", domainPrograms[1])
	}
}

func TestToDomainProgramsIsNewIsMovie(t *testing.T) {
	now := time.Now()

	t.Run("detects new programs", func(t *testing.T) {
		programs := []XMLTVProgram{
			{ChannelID: "ch1", StartTime: now, EndTime: now, Title: "New Episode", Category: "New"},
			{ChannelID: "ch1", StartTime: now, EndTime: now, Title: "Show (U)"},
			{ChannelID: "ch1", StartTime: now, EndTime: now, Title: "Regular", Desc: "has (U) in desc"},
		}
		epgToChannelID := map[string]int64{"ch1": 1}
		result := ToDomainPrograms(programs, epgToChannelID)
		if len(result) != 3 {
			t.Fatalf("expected 3 programs, got %d", len(result))
		}
		if !result[0].IsNew {
			t.Error("program 0 should be IsNew (category contains 'New')")
		}
		if !result[1].IsNew {
			t.Error("program 1 should be IsNew (title contains '(U)')")
		}
		if !result[2].IsNew {
			t.Error("program 2 should be IsNew (description contains '(U)')")
		}
	})

	t.Run("detects movie programs", func(t *testing.T) {
		programs := []XMLTVProgram{
			{ChannelID: "ch1", StartTime: now, EndTime: now, Title: "Film", Category: "Movie"},
			{ChannelID: "ch1", StartTime: now, EndTime: now, Title: "Documentary", Category: "Film"},
		}
		epgToChannelID := map[string]int64{"ch1": 1}
		result := ToDomainPrograms(programs, epgToChannelID)
		if !result[0].IsMovie {
			t.Error("program 0 should be IsMovie")
		}
		if !result[1].IsMovie {
			t.Error("program 1 should be IsMovie")
		}
	})
}
