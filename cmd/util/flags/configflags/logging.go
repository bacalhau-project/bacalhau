package configflags

import (
	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
)

var LogFlags = []Definition{
	{
		FlagName:     "log-mode",
		DefaultValue: types2.Default.Logging.Mode,
		ConfigPath:   "Logging.Mode",
		Description:  `Log format: 'default','station','json','combined','event'`,
	},
	{
		FlagName:             "log-level",
		DefaultValue:         types2.Default.Logging.Level,
		ConfigPath:           "Logging.Level",
		Description:          `Log level: 'trace', 'debug', 'info', 'warn', 'error', 'fatal', 'panic'`,
		EnvironmentVariables: []string{"LOG_LEVEL"},
	},
}
