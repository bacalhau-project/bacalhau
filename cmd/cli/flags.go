package cli

import (
	"github.com/bacalhau-project/bacalhau/cmd/util/flags"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
)

var APIFlags = []flags.FlagDefinition{
	{
		FlagName:     "api-host",
		DefaultValue: config.Default.Node.API.Host,
		ConfigPath:   config.NodeAPIHost,
		Description: `The host for the client and server to communicate on (via REST).
Ignored if BACALHAU_API_HOST environment variable is set.`,
	},
	{
		FlagName:     "api-port",
		DefaultValue: config.Default.Node.API.Port,
		ConfigPath:   config.NodeAPIPort,
		Description: `The port for the client and server to communicate on (via REST).
Ignored if BACALHAU_API_PORT environment variable is set.`,
	},
}

var LogFlags = []flags.FlagDefinition{
	{
		FlagName:     "log-mode",
		DefaultValue: logger.LogMode(logger.LogModeDefault),
		ConfigPath:   config.NodeLoggingMode,
		Description:  `Log format: 'default','station','json','combined','event'`,
	},
}
