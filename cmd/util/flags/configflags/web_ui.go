package configflags

import "github.com/bacalhau-project/bacalhau/pkg/config/types"

var WebUIFlags = []Definition{
	{
		FlagName:     "web-ui",
		ConfigPath:   types.WebUI,
		DefaultValue: false,
		Description:  `Whether to start the web UI alongside the bacalhau node.`,
	},
}
