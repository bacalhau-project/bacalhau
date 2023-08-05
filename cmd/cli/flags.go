package cli

import (
	"github.com/bacalhau-project/bacalhau/cmd/util/flags"
	"github.com/bacalhau-project/bacalhau/pkg/config_v2"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
)

var APIFlags = []flags.FlagDefinition{
	{
		FlagName:     "api-host",
		DefaultValue: config_v2.Default.Node.API.Host,
		ConfigPath:   config_v2.NodeAPIHost,
		Description: `The host for the client and server to communicate on (via REST).
Ignored if BACALHAU_API_HOST environment variable is set.`,
	},
	{
		FlagName:     "api-port",
		DefaultValue: config_v2.Default.Node.API.Port,
		ConfigPath:   config_v2.NodeAPIPort,
		Description: `The port for the client and server to communicate on (via REST).
Ignored if BACALHAU_API_PORT environment variable is set.`,
	},
}

var LogFlags = []flags.FlagDefinition{
	{
		FlagName:     "log-mode",
		DefaultValue: logger.LogModeDefault,
		ConfigPath:   config_v2.NodeLoggingMode,
		Description:  `Log format: 'default','station','json','combined','event'`,
	},
}
