package livetv

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/mantonx/viewra/internal/domain/livetv"
	"github.com/mantonx/viewra/internal/infrastructure/database/unified"
	"github.com/mantonx/viewra/internal/infrastructure/persistence/common"
)

func NewChannelRepository(baseRepo *common.BaseRepository) *ChannelRepository {
	return &ChannelRepository{BaseRepository: baseRepo}
}

func NewProgramRepository(baseRepo *common.BaseRepository) *ProgramRepository {
	return &ProgramRepository{BaseRepository: baseRepo}
}

func liveChannelToDomain(c unified.LiveChannel) *livetv.Channel {
	return &livetv.Channel{
		ID:            c.ID,
		LibraryID:     c.LibraryID,
		ChannelNumber: int(c.ChannelNumber),
		Name:          c.Name,
		StreamURL:     c.StreamUrl,
		LogoURL:       common.ParseNullString(c.LogoUrl),
		Group:         common.ParseNullString(c.ChannelGroup),
		EPGChannelID:  common.ParseNullString(c.EpgChannelID),
		Enabled:       common.Int64ToBool(c.Enabled),
		CreatedAt:     common.ParseNullTime(c.CreatedAt),
		UpdatedAt:     common.ParseNullTime(c.UpdatedAt),
	}
}

func liveEpgProgramToDomain(p unified.LiveEpgProgram) *livetv.Program {
	return &livetv.Program{
		ID:           p.ID,
		ChannelID:    p.ChannelID,
		StartTime:    p.StartTime,
		EndTime:      p.EndTime,
		Title:        p.Title,
		SubTitle:     common.ParseNullString(p.SubTitle),
		Description:  common.ParseNullString(p.Description),
		Category:     common.ParseNullString(p.Category),
		EpisodeTitle: common.ParseNullString(p.EpisodeTitle),
		EpisodeNum:   int(common.ParseNullInt64(p.EpisodeNum)),
		SeasonNum:    int(common.ParseNullInt64(p.SeasonNum)),
		IsNew:        common.NullInt64ToBool(p.IsNew),
		IsMovie:      common.NullInt64ToBool(p.IsMovie),
		CreatedAt:    common.ParseNullTime(p.CreatedAt),
	}
}

func (r *ChannelRepository) Create(ctx context.Context, channel *livetv.Channel) error {
	result, err := r.Q().CreateChannel(ctx, unified.CreateChannelParams{
		LibraryID:     channel.LibraryID,
		ChannelNumber: int64(channel.ChannelNumber),
		Name:          channel.Name,
		StreamUrl:     channel.StreamURL,
		LogoUrl:       common.NullString(channel.LogoURL),
		ChannelGroup:  common.NullString(channel.Group),
		EpgChannelID:  common.NullString(channel.EPGChannelID),
		Enabled:       common.BoolToInt64(channel.Enabled),
	})
	if err != nil {
		return err
	}
	channel.ID = result.ID
	channel.CreatedAt = common.ParseNullTime(result.CreatedAt)
	channel.UpdatedAt = common.ParseNullTime(result.UpdatedAt)
	return nil
}

func (r *ChannelRepository) GetByID(ctx context.Context, id int64) (*livetv.Channel, error) {
	result, err := r.Q().GetChannel(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, livetv.ErrChannelNotFound
		}
		return nil, err
	}
	return liveChannelToDomain(result), nil
}

func (r *ChannelRepository) ListByLibrary(ctx context.Context, libraryID int64) ([]*livetv.Channel, error) {
	results, err := r.Q().ListChannels(ctx, libraryID)
	if err != nil {
		return nil, err
	}
	return mapSlice(results, liveChannelToDomain), nil
}

func (r *ChannelRepository) Upsert(ctx context.Context, channel *livetv.Channel) error {
	result, err := r.Q().UpsertChannel(ctx, unified.UpsertChannelParams{
		LibraryID:     channel.LibraryID,
		ChannelNumber: int64(channel.ChannelNumber),
		Name:          channel.Name,
		StreamUrl:     channel.StreamURL,
		LogoUrl:       common.NullString(channel.LogoURL),
		ChannelGroup:  common.NullString(channel.Group),
		EpgChannelID:  common.NullString(channel.EPGChannelID),
		Enabled:       common.BoolToInt64(channel.Enabled),
	})
	if err != nil {
		return err
	}
	channel.ID = result.ID
	channel.CreatedAt = common.ParseNullTime(result.CreatedAt)
	channel.UpdatedAt = common.ParseNullTime(result.UpdatedAt)
	return nil
}

func (r *ChannelRepository) Delete(ctx context.Context, id int64) error {
	return r.Q().DeleteChannel(ctx, id)
}

func (r *ChannelRepository) DeleteByLibrary(ctx context.Context, libraryID int64) error {
	return r.Q().DeleteChannelsByLibrary(ctx, libraryID)
}

func (r *ProgramRepository) Create(ctx context.Context, program *livetv.Program) error {
	result, err := r.Q().CreateProgram(ctx, unified.CreateProgramParams{
		ChannelID:    program.ChannelID,
		StartTime:    program.StartTime,
		EndTime:      program.EndTime,
		Title:        program.Title,
		SubTitle:     common.NullString(program.SubTitle),
		Description:  common.NullString(program.Description),
		Category:     common.NullString(program.Category),
		EpisodeTitle: common.NullString(program.EpisodeTitle),
		EpisodeNum:   common.NullInt64(int64(program.EpisodeNum)),
		SeasonNum:    common.NullInt64(int64(program.SeasonNum)),
		IsNew:        common.NullInt64FromBool(program.IsNew),
		IsMovie:      common.NullInt64FromBool(program.IsMovie),
	})
	if err != nil {
		return err
	}
	program.ID = result.ID
	program.CreatedAt = common.ParseNullTime(result.CreatedAt)
	return nil
}

func (r *ProgramRepository) GetByID(ctx context.Context, id int64) (*livetv.Program, error) {
	result, err := r.Q().GetProgram(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, livetv.ErrProgramNotFound
		}
		return nil, err
	}
	return liveEpgProgramToDomain(result), nil
}

func (r *ProgramRepository) ListByChannel(ctx context.Context, channelID int64, from, to time.Time) ([]*livetv.Program, error) {
	results, err := r.Q().ListProgramsByChannel(ctx, unified.ListProgramsByChannelParams{
		ChannelID: channelID,
		StartTime: to,
		EndTime:   from,
	})
	if err != nil {
		return nil, err
	}
	return mapSlice(results, liveEpgProgramToDomain), nil
}

func (r *ProgramRepository) ListByLibrary(ctx context.Context, libraryID int64, from, to time.Time) ([]*livetv.Program, error) {
	results, err := r.Q().ListProgramsByLibrary(ctx, unified.ListProgramsByLibraryParams{
		LibraryID: libraryID,
		StartTime: to,
		EndTime:   from,
	})
	if err != nil {
		return nil, err
	}
	return mapSlice(results, liveEpgProgramToDomain), nil
}

func (r *ProgramRepository) GetCurrentByChannel(ctx context.Context, channelID int64, now time.Time) (*livetv.Program, error) {
	result, err := r.Q().GetCurrentProgram(ctx, unified.GetCurrentProgramParams{
		ChannelID: channelID,
		StartTime: now,
		EndTime:   now,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, livetv.ErrProgramNotFound
		}
		return nil, err
	}
	return liveEpgProgramToDomain(result), nil
}

func (r *ProgramRepository) DeleteByChannel(ctx context.Context, channelID int64) error {
	return r.Q().DeleteProgramsByChannel(ctx, channelID)
}

func (r *ProgramRepository) DeleteByLibrary(ctx context.Context, libraryID int64) error {
	return r.Q().DeleteProgramsByLibrary(ctx, libraryID)
}

func (r *ProgramRepository) BulkCreate(ctx context.Context, programs []*livetv.Program) error {
	for _, p := range programs {
		err := r.Q().BulkCreatePrograms(ctx, unified.BulkCreateProgramsParams{
			ChannelID:    p.ChannelID,
			StartTime:    p.StartTime,
			EndTime:      p.EndTime,
			Title:        p.Title,
			SubTitle:     common.NullString(p.SubTitle),
			Description:  common.NullString(p.Description),
			Category:     common.NullString(p.Category),
			EpisodeTitle: common.NullString(p.EpisodeTitle),
			EpisodeNum:   common.NullInt64(int64(p.EpisodeNum)),
			SeasonNum:    common.NullInt64(int64(p.SeasonNum)),
			IsNew:        common.NullInt64FromBool(p.IsNew),
			IsMovie:      common.NullInt64FromBool(p.IsMovie),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func mapSlice[TFrom, TTo any](from []TFrom, mapper func(TFrom) TTo) []TTo {
	result := make([]TTo, len(from))
	for i, v := range from {
		result[i] = mapper(v)
	}
	return result
}
