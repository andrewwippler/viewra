package livetv

import "time"

// Program represents an EPG program entry from XMLTV data
type Program struct {
	ID           int64
	ChannelID    int64
	StartTime    time.Time
	EndTime      time.Time
	Title        string
	SubTitle     string
	Description  string
	Category     string
	EpisodeTitle string
	EpisodeNum   int
	SeasonNum    int
	IsNew        bool
	IsMovie      bool
	CreatedAt    time.Time
}
