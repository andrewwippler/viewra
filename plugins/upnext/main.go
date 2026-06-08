package main

import (
	"github.com/mantonx/viewra/pkg/plugin/sdk"
	"github.com/mantonx/viewra/plugins/upnext/internal"
)

func main() {
	hclogger, logger := sdk.NewLogger("upnext")
	plugin := internal.NewUpNextPlugin(logger)
	sdk.ServeWidget(plugin, hclogger)
}
