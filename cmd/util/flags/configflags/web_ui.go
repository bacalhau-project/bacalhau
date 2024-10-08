package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

var WebUIFlags = []Definition{
	{
		FlagName:          "web-ui",
		ConfigPath:        types.WebUIEnabledKey,
		DefaultValue:      config.Default.WebUI.Enabled,
		Description:       `Whether to start the web UI alongside the bacalhau node.`,
		Deprecated:        true,
		DeprecatedMessage: makeDeprecationMessage(types.WebUIEnabledKey),
	},
	{
		FlagName:          "web-ui-listen",
		ConfigPath:        types.WebUIListenKey,
		DefaultValue:      config.Default.WebUI.Listen,
		Description:       `The address to listen on for web-ui connections.`,
		Deprecated:        true,
		DeprecatedMessage: makeDeprecationMessage(types.WebUIListenKey),
	},
}
