package configflags

import (
	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
)

var DisabledFeatureFlags = []Definition{
	{
		FlagName:     "disable-engine",
		ConfigPath:   "Executors.Disabled",
		DefaultValue: types2.Default.Executors.Disabled,
		Description:  "Engine types to disable",
	},
	{
		FlagName:     "disabled-publisher",
		ConfigPath:   "Publishers.Disabled",
		DefaultValue: types2.Default.Publishers.Disabled,
		Description:  "Publisher types to disable",
	},
	{
		FlagName:     "disable-storage",
		ConfigPath:   "InputSources.Disabled",
		DefaultValue: types2.Default.InputSources.Disabled,
		Description:  "Storage types to disable",
	},
}
