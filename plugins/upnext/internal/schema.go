package internal

import (
	"github.com/mantonx/viewra/pkg/plugin/sdk"
)

func SettingsSchema() *sdk.Schema {
	return sdk.NewSchema("Up Next Settings").
		Meta(sdk.PluginMeta{
			DisplayName: "Up Next",
			Description: "Shows the next unwatched TV episode for shows you're watching",
			Tip:         "Mark episodes as watched to track your progress.",
			Icon:        "tv",
		}).
		Property("enabled", sdk.Boolean().
			Title("Enable Up Next").
			Description("Show upcoming episodes on the home screen").
			Default(true)).
		Widgets([]sdk.Widget{
			{
				ID:              "up-next",
				Type:            sdk.WidgetTypeContinueRow,
				Location:        sdk.LocationHomepageSections,
				ClientTypes:     []string{sdk.ClientTypeAll},
				Priority:        80,
				CacheTTLSeconds: 300,
				Config: map[string]any{
					"endpoint": "/up-next",
					"title":    "Up Next",
				},
				SettingsKey: "enabled",
			},
		})
}
