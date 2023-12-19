package configflags

import "github.com/bacalhau-project/bacalhau/pkg/config/types"

var WebUIFlags = []Definition{
	{
		FlagName:     "web-ui",
		ConfigPath:   types.NodeWebUIEnabled,
		DefaultValue: false,
		Description:  `Whether to start the web UI alongside the bacalhau node.`,
	},
	{
		FlagName:     "web-ui-port",
		ConfigPath:   types.NodeWebUIPort,
		DefaultValue: 80,
		Description:  `The port number to listen on for web-ui connections.`,
	},
}
