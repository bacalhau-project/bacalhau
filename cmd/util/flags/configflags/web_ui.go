package configflags

import (
	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
)

var WebUIFlags = []Definition{
	{
		FlagName:     "web-ui",
		ConfigPath:   "WebUI.Enabled",
		DefaultValue: types2.Default.WebUI.Enabled,
		Description:  `Whether to start the web UI alongside the bacalhau node.`,
	},
	{
		FlagName:     "web-ui-listen",
		ConfigPath:   "WebUI.Listen",
		DefaultValue: types2.Default.WebUI.Listen,
		Description:  `The address to listen on for web-ui connections.`,
	},
}
