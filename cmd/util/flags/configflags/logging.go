package configflags

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

// TODO: remove this once we have a proper way to register LOG_LEVEL env var into the config.
//
//	Currently we utilize cli flags to also register env vars.
var LogFlags = []Definition{
	{
		FlagName:             "log-level",
		DefaultValue:         config.Default.Logging.Level,
		ConfigPath:           types.LoggingLevelKey,
		Description:          `Log level: 'trace', 'debug', 'info', 'warn', 'error', 'fatal', 'panic'`,
		EnvironmentVariables: []string{"LOG_LEVEL"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.LoggingLevelKey),
	},
}

func makeDeprecationMessage(key string) string {
	return fmt.Sprintf("Use %s to set this configuration", makeConfigFlagDeprecationCommand(key))
}

func makeConfigFlagDeprecationCommand(key string) string {
	return fmt.Sprintf("--config %s=<value>", key)
}
