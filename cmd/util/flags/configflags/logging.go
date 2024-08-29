package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/cfgtypes"
)

var LogFlags = []Definition{
	{
		FlagName:     "log-mode",
		DefaultValue: cfgtypes.Default.Logging.Mode,
		ConfigPath:   cfgtypes.LoggingModeKey,
		Description:  `Log format: 'default','station','json','combined','event'`,
	},
	{
		FlagName:             "log-level",
		DefaultValue:         cfgtypes.Default.Logging.Level,
		ConfigPath:           cfgtypes.LoggingLevelKey,
		Description:          `Log level: 'trace', 'debug', 'info', 'warn', 'error', 'fatal', 'panic'`,
		EnvironmentVariables: []string{"LOG_LEVEL"},
	},
}
