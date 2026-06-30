package livetv

import (
	"time"
)

// Channel represents a live TV channel from an M3U playlist
type Channel struct {
	ID           int64
	LibraryID    int64
	ChannelNumber int
	Name         string
	StreamURL    string
	LogoURL      string
	Group        string
	EPGChannelID string
	Enabled      bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
