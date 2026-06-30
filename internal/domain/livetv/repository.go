package livetv

import (
	"context"
	"time"
)

// ChannelRepository defines persistence operations for live TV channels
type ChannelRepository interface {
	Create(ctx context.Context, channel *Channel) error
	GetByID(ctx context.Context, id int64) (*Channel, error)
	ListByLibrary(ctx context.Context, libraryID int64) ([]*Channel, error)
	Upsert(ctx context.Context, channel *Channel) error
	Delete(ctx context.Context, id int64) error
	DeleteByLibrary(ctx context.Context, libraryID int64) error
}

// ProgramRepository defines persistence operations for EPG programs
type ProgramRepository interface {
	Create(ctx context.Context, program *Program) error
	GetByID(ctx context.Context, id int64) (*Program, error)
	ListByChannel(ctx context.Context, channelID int64, from, to time.Time) ([]*Program, error)
	ListByLibrary(ctx context.Context, libraryID int64, from, to time.Time) ([]*Program, error)
	GetCurrentByChannel(ctx context.Context, channelID int64, now time.Time) (*Program, error)
	DeleteByChannel(ctx context.Context, channelID int64) error
	DeleteByLibrary(ctx context.Context, libraryID int64) error
	BulkCreate(ctx context.Context, programs []*Program) error
}
