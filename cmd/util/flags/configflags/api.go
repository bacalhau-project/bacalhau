package configflags

import "github.com/bacalhau-project/bacalhau/pkg/config/types"

var APIFlags = []Definition{
	{
		FlagName:     "api-host",
		DefaultValue: Default.Node.API.Host,
		ConfigPath:   types.NodeAPIHost,
		Description: `The host for the client and server to communicate on (via REST).
Ignored if BACALHAU_API_HOST environment variable is set.`,
	},
	{
		FlagName:     "api-port",
		DefaultValue: Default.Node.API.Port,
		ConfigPath:   types.NodeAPIPort,
		Description: `The port for the client and server to communicate on (via REST).
Ignored if BACALHAU_API_PORT environment variable is set.`,
	},
}
