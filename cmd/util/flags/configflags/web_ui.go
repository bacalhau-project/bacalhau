package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

var WebUIFlags = []Definition{
	{
		FlagName:     "web-ui",
		ConfigPath:   types.WebUIEnabledKey,
		DefaultValue: types.Default.WebUI.Enabled,
		Description:  `Whether to start the web UI alongside the bacalhau node.`,
	},
	{
		FlagName:     "web-ui-listen",
		ConfigPath:   types.WebUIListenKey,
		DefaultValue: types.Default.WebUI.Listen,
		Description:  `The address to listen on for web-ui connections.`,
	},
}
