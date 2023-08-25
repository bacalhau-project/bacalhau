package configflags

import "github.com/bacalhau-project/bacalhau/pkg/config/types"

var ClientAPIFlags = []Definition{
	{
		FlagName:     "api-host",
		DefaultValue: Default.Node.ClientAPI.Host,
		ConfigPath:   types.NodeClientAPIHost,
		Description: `The host for the client and server to communicate on (via REST).
Ignored if BACALHAU_API_HOST environment variable is set.`,
		EnvironmentVariables: []string{"BACALHAU_HOST"},
	},
	{
		FlagName:     "api-port",
		DefaultValue: Default.Node.ClientAPI.Port,
		ConfigPath:   types.NodeClientAPIPort,
		Description: `The port for the client and server to communicate on (via REST).
Ignored if BACALHAU_API_PORT environment variable is set.`,
		EnvironmentVariables: []string{"BACALHAU_PORT"},
	},
}

var ServerAPIFlags = []Definition{
	{
		FlagName:             "server-api-port",
		DefaultValue:         Default.Node.ServerAPI.Port,
		ConfigPath:           types.NodeServerAPIPort,
		Description:          `The port to server on.`,
		EnvironmentVariables: []string{"BACALHAU_PORT"},
	},
	{
		FlagName:     "server-api-host",
		DefaultValue: Default.Node.ServerAPI.Host,
		ConfigPath:   types.NodeServerAPIHost,
		Description:  `The host to serve on.`,
	},
}
