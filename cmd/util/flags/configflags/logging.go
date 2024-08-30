package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

var LogFlags = []Definition{
	{
		FlagName:     "log-mode",
		DefaultValue: types.Default.Logging.Mode,
		ConfigPath:   types.LoggingModeKey,
		Description:  `Log format: 'default','station','json','combined','event'`,
	},
	{
		FlagName:             "log-level",
		DefaultValue:         types.Default.Logging.Level,
		ConfigPath:           types.LoggingLevelKey,
		Description:          `Log level: 'trace', 'debug', 'info', 'warn', 'error', 'fatal', 'panic'`,
		EnvironmentVariables: []string{"LOG_LEVEL"},
	},
}
