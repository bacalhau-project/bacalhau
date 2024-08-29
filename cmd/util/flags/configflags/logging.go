package configflags

import (
	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
)

var LogFlags = []Definition{
	{
		FlagName:     "log-mode",
		DefaultValue: types2.Default.Logging.Mode,
		ConfigPath:   types2.LoggingModeKey,
		Description:  `Log format: 'default','station','json','combined','event'`,
	},
	{
		FlagName:             "log-level",
		DefaultValue:         types2.Default.Logging.Level,
		ConfigPath:           types2.LoggingLevelKey,
		Description:          `Log level: 'trace', 'debug', 'info', 'warn', 'error', 'fatal', 'panic'`,
		EnvironmentVariables: []string{"LOG_LEVEL"},
	},
}
