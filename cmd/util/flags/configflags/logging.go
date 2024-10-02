package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

var LogFlags = []Definition{
	{
		FlagName:          "log-mode",
		DefaultValue:      config.Default.Logging.Mode,
		ConfigPath:        types.LoggingModeKey,
		Description:       `Log format: 'default','station','json','combined','event'`,
		Deprecated:        true,
		DeprecatedMessage: makeDeprecationMessage(types.LoggingModeKey),
	},
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
