package livetv

import (
	"errors"
	"testing"
	"time"
)

func TestChannelCreation(t *testing.T) {
	now := time.Now()
	ch := &Channel{
		ID:            1,
		LibraryID:     42,
		ChannelNumber: 5,
		Name:          "Test Channel",
		StreamURL:     "http://stream.example.com/test.m3u8",
		LogoURL:       "http://logo.tv/test.png",
		Group:         "News",
		EPGChannelID:  "test_channel",
		Enabled:       true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if ch.ID != 1 {
		t.Errorf("ID = %d", ch.ID)
	}
	if ch.LibraryID != 42 {
		t.Errorf("LibraryID = %d", ch.LibraryID)
	}
	if ch.ChannelNumber != 5 {
		t.Errorf("ChannelNumber = %d", ch.ChannelNumber)
	}
	if ch.Name != "Test Channel" {
		t.Errorf("Name = %q", ch.Name)
	}
}

func TestProgramCreation(t *testing.T) {
	start := time.Date(2026, 6, 13, 4, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 13, 5, 0, 0, 0, time.UTC)
	p := &Program{
		ID:           1,
		ChannelID:    100,
		StartTime:    start,
		EndTime:      end,
		Title:        "Morning News",
		SubTitle:     "Weekend Edition",
		Description:  "Latest headlines",
		Category:     "News",
		EpisodeTitle: "Episode 1",
		EpisodeNum:   1,
		SeasonNum:    1,
		IsNew:        true,
		IsMovie:      false,
		CreatedAt:    time.Now(),
	}
	if p.Title != "Morning News" {
		t.Errorf("Title = %q", p.Title)
	}
	if !p.StartTime.Equal(start) {
		t.Errorf("StartTime = %v", p.StartTime)
	}
	if !p.EndTime.Equal(end) {
		t.Errorf("EndTime = %v", p.EndTime)
	}
	if !p.IsNew {
		t.Error("IsNew should be true")
	}
	if p.IsMovie {
		t.Error("IsMovie should be false")
	}
}

func TestErrorSentinels(t *testing.T) {
	if !errors.Is(ErrChannelNotFound, ErrChannelNotFound) {
		t.Error("ErrChannelNotFound should match itself")
	}
	if !errors.Is(ErrProgramNotFound, ErrProgramNotFound) {
		t.Error("ErrProgramNotFound should match itself")
	}
	if !errors.Is(ErrInvalidChannel, ErrInvalidChannel) {
		t.Error("ErrInvalidChannel should match itself")
	}
	if !errors.Is(ErrInvalidProgram, ErrInvalidProgram) {
		t.Error("ErrInvalidProgram should match itself")
	}

	if errors.Is(ErrChannelNotFound, ErrProgramNotFound) {
		t.Error("ErrChannelNotFound should NOT match ErrProgramNotFound")
	}
	if errors.Is(ErrInvalidChannel, ErrChannelNotFound) {
		t.Error("ErrInvalidChannel should NOT match ErrChannelNotFound")
	}
}
