package livetv

import (
	"context"

	"github.com/mantonx/viewra/internal/application/library"
	"github.com/mantonx/viewra/internal/application/scheduler"
)

// Tasks exports all scheduled tasks for the live tv domain.
var Tasks = []scheduler.TaskBuilder{
	&epgRefreshTask{},
	&satipPIDSyncTask{},
}

type epgRefreshTask struct{}

func (t *epgRefreshTask) Definition() scheduler.TaskDefinition {
	return scheduler.TaskDefinition{
		ID:             "internal:livetv:epg-refresh",
		Name:           "EPG Data Refresh",
		Description:    "Refresh EPG program data from XMLTV URLs for all live TV libraries",
		Schedule:       "0 4 * * *", // Daily at 4 AM
		Group:          "livetv",
		TimeoutSeconds: 600,
	}
}

type epgScanner interface {
	Execute(ctx context.Context, libraryID int64) (ScanResponse, error)
}

func (t *epgRefreshTask) Build(deps *scheduler.RuntimeDeps) func(context.Context) error {
	libService := deps.LibraryLister.(*library.LibraryService)
	scanner := deps.EPGScanner.(epgScanner)

	return func(ctx context.Context) error {
		resp, err := libService.List(ctx)
		if err != nil {
			return err
		}

		refreshed := 0
		for _, lib := range resp.Libraries {
			if lib.Type != "live_tv" {
				continue
			}

			if lib.MonitoringConfig == nil || lib.MonitoringConfig.XmltvURL == "" {
				continue
			}

			_, err := scanner.Execute(ctx, lib.ID)
			if err != nil {
				deps.Logger.Error("EPG refresh failed for library",
					"library_id", lib.ID,
					"library_name", lib.Name,
					"error", err,
				)
				continue
			}
			refreshed++
		}

		deps.Logger.Info("EPG refresh completed", "libraries_refreshed", refreshed)
		return nil
	}
}
