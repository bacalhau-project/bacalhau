package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/cfgtypes"
)

var DisabledFeatureFlags = []Definition{
	{
		FlagName:     "disable-engine",
		ConfigPath:   cfgtypes.EnginesDisabledKey,
		DefaultValue: cfgtypes.Default.Engines.Disabled,
		Description:  "Engine types to disable",
	},
	{
		FlagName:     "disabled-publisher",
		ConfigPath:   cfgtypes.PublishersDisabledKey,
		DefaultValue: cfgtypes.Default.Publishers.Disabled,
		Description:  "Publisher types to disable",
	},
	{
		FlagName:     "disable-storage",
		ConfigPath:   cfgtypes.InputSourcesDisabledKey,
		DefaultValue: cfgtypes.Default.InputSources.Disabled,
		Description:  "Storage types to disable",
	},
}
