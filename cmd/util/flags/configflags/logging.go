package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
)

var LogFlags = []Definition{
	{
		FlagName:     "log-mode",
		DefaultValue: logger.LogModeDefault,
		ConfigPath:   types.NodeLoggingMode,
		Description:  `Log format: 'default','station','json','combined','event'`,
	},
}
