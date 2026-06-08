package main

import (
	"github.com/mantonx/viewra/pkg/plugin/sdk"
	"github.com/mantonx/viewra/plugins/nitpicky-edits/internal"
)

func main() {
	hclogger, logger := sdk.NewLogger("nitpicky-edits")
	plugin := internal.NewPlugin(logger)
	sdk.ServeEnricher(plugin, hclogger)
}
