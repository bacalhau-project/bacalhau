package configflags

import (
	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
)

var DisabledFeatureFlags = []Definition{
	{
		FlagName:     "disable-engine",
		ConfigPath:   types2.EnginesDisabledKey,
		DefaultValue: types2.Default.Engines.Disabled,
		Description:  "Engine types to disable",
	},
	{
		FlagName:     "disabled-publisher",
		ConfigPath:   types2.PublishersDisabledKey,
		DefaultValue: types2.Default.Publishers.Disabled,
		Description:  "Publisher types to disable",
	},
	{
		FlagName:     "disable-storage",
		ConfigPath:   types2.InputSourcesDisabledKey,
		DefaultValue: types2.Default.InputSources.Disabled,
		Description:  "Storage types to disable",
	},
}
