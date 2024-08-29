package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/cfgtypes"
)

var WebUIFlags = []Definition{
	{
		FlagName:     "web-ui",
		ConfigPath:   cfgtypes.WebUIEnabledKey,
		DefaultValue: cfgtypes.Default.WebUI.Enabled,
		Description:  `Whether to start the web UI alongside the bacalhau node.`,
	},
	{
		FlagName:     "web-ui-listen",
		ConfigPath:   cfgtypes.WebUIListenKey,
		DefaultValue: cfgtypes.Default.WebUI.Listen,
		Description:  `The address to listen on for web-ui connections.`,
	},
}
