package livetv

import (
	"context"
	"fmt"
	"time"

	"github.com/mantonx/viewra/internal/domain/livetv"
)

type ListChannelsUseCase struct {
	channelRepo livetv.ChannelRepository
	programRepo livetv.ProgramRepository
}

func NewListChannelsUseCase(channelRepo livetv.ChannelRepository, programRepo livetv.ProgramRepository) *ListChannelsUseCase {
	return &ListChannelsUseCase{
		channelRepo: channelRepo,
		programRepo: programRepo,
	}
}

func (uc *ListChannelsUseCase) Execute(ctx context.Context, libraryID int64) (ListChannelsResponse, error) {
	channels, err := uc.channelRepo.ListByLibrary(ctx, libraryID)
	if err != nil {
		return ListChannelsResponse{}, fmt.Errorf("failed to list channels: %w", err)
	}

	now := time.Now()
	resp := make([]ChannelResponse, len(channels))
	for i, ch := range channels {
		r := toChannelResponse(ch)
		prog, err := uc.programRepo.GetCurrentByChannel(ctx, ch.ID, now)
		if err == nil {
			p := toProgramResponse(prog)
			r.CurrentProgram = &p
		}
		resp[i] = r
	}

	return ListChannelsResponse{Channels: resp}, nil
}
