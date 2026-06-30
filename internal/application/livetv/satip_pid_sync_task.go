package livetv

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/mantonx/viewra/internal/application/library"
	"github.com/mantonx/viewra/internal/application/scheduler"
	"github.com/mantonx/viewra/internal/domain/livetv"
	infraLivetv "github.com/mantonx/viewra/internal/infrastructure/livetv"
)

type satipPIDSyncTask struct{}

func (t *satipPIDSyncTask) Definition() scheduler.TaskDefinition {
	return scheduler.TaskDefinition{
		ID:             "internal:livetv:satip-pid-sync",
		Name:           "SAT>IP PID Sync",
		Description:    "Synchronize SAT>IP channel PIDs from the authoritative channellist for all live TV libraries",
		Schedule:       "0 3 1 * *", // Monthly on the 1st at 3 AM
		Group:          "livetv",
		TimeoutSeconds: 600,
	}
}

type liveTvChannelRepo interface {
	ListByLibrary(ctx context.Context, libraryID int64) ([]*livetv.Channel, error)
	Upsert(ctx context.Context, channel *livetv.Channel) error
}

func (t *satipPIDSyncTask) Build(deps *scheduler.RuntimeDeps) func(context.Context) error {
	libService := deps.LibraryLister.(*library.LibraryService)
	channelRepo := deps.LiveTvChannelRepo.(liveTvChannelRepo)
	httpClient := &http.Client{Timeout: 30 * time.Second}

	return func(ctx context.Context) error {
		resp, err := libService.List(ctx)
		if err != nil {
			return fmt.Errorf("failed to list libraries: %w", err)
		}

		corrected := 0
		checked := 0

		for _, lib := range resp.Libraries {
			if lib.Type != "live_tv" {
				continue
			}

			if lib.MonitoringConfig == nil || lib.MonitoringConfig.SatipChannelListURL == "" {
				continue
			}

			checked++

			freqToPIDs, err := infraLivetv.FetchSatipChannelList(ctx, lib.MonitoringConfig.SatipChannelListURL, httpClient)
			if err != nil {
				deps.Logger.Error("failed to fetch SAT>IP channellist",
					"library_id", lib.ID,
					"library_name", lib.Name,
					"error", err,
				)
				continue
			}

			channels, err := channelRepo.ListByLibrary(ctx, lib.ID)
			if err != nil {
				deps.Logger.Error("failed to list channels",
					"library_id", lib.ID,
					"library_name", lib.Name,
					"error", err,
				)
				continue
			}

			for _, ch := range channels {
				if infraLivetv.CorrectChannelPIDsFromList(ch, freqToPIDs) {
					if err := channelRepo.Upsert(ctx, ch); err != nil {
						deps.Logger.Error("failed to upsert corrected channel",
							"channel_id", ch.ID,
							"channel_name", ch.Name,
							"error", err,
						)
						continue
					}
					corrected++
				}
			}
		}

		deps.Logger.Info("SAT>IP PID sync completed",
			"libraries_checked", checked,
			"channels_corrected", corrected,
		)
		return nil
	}
}
