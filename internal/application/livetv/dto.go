package livetv

import (
	"time"

	"github.com/mantonx/viewra/internal/domain/livetv"
)

type ChannelResponse struct {
	ID            int64  `json:"id"`
	LibraryID     int64  `json:"library_id"`
	ChannelNumber int    `json:"channel_number"`
	Name          string `json:"name"`
	StreamURL     string `json:"stream_url"`
	LogoURL       string `json:"logo_url,omitempty"`
	Group         string `json:"group,omitempty"`
	EPGChannelID  string `json:"epg_channel_id,omitempty"`
	Enabled       bool   `json:"enabled"`
	CurrentProgram *ProgramResponse `json:"current_program,omitempty"`
}

type ProgramResponse struct {
	ID           int64     `json:"id"`
	ChannelID    int64     `json:"channel_id"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Title        string    `json:"title"`
	SubTitle     string    `json:"sub_title,omitempty"`
	Description  string    `json:"description,omitempty"`
	Category     string    `json:"category,omitempty"`
	EpisodeTitle string    `json:"episode_title,omitempty"`
	EpisodeNum   int       `json:"episode_num,omitempty"`
	SeasonNum    int       `json:"season_num,omitempty"`
	IsNew        bool      `json:"is_new,omitempty"`
	IsMovie      bool      `json:"is_movie,omitempty"`
}

type ListChannelsResponse struct {
	Channels []ChannelResponse `json:"channels"`
}

type ListEPGResponse struct {
	Programs []ProgramResponse `json:"programs"`
}

type ScanResponse struct {
	Message string `json:"message"`
}

func toChannelResponse(ch *livetv.Channel) ChannelResponse {
	return ChannelResponse{
		ID:            ch.ID,
		LibraryID:     ch.LibraryID,
		ChannelNumber: ch.ChannelNumber,
		Name:          ch.Name,
		StreamURL:     ch.StreamURL,
		LogoURL:       ch.LogoURL,
		Group:         ch.Group,
		EPGChannelID:  ch.EPGChannelID,
		Enabled:       ch.Enabled,
	}
}

func toProgramResponse(p *livetv.Program) ProgramResponse {
	return ProgramResponse{
		ID:           p.ID,
		ChannelID:    p.ChannelID,
		StartTime:    p.StartTime,
		EndTime:      p.EndTime,
		Title:        p.Title,
		SubTitle:     p.SubTitle,
		Description:  p.Description,
		Category:     p.Category,
		EpisodeTitle: p.EpisodeTitle,
		EpisodeNum:   p.EpisodeNum,
		SeasonNum:    p.SeasonNum,
		IsNew:        p.IsNew,
		IsMovie:      p.IsMovie,
	}
}
