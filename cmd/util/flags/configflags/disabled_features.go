package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

var DisabledFeatureFlags = []Definition{
	{
		FlagName:          "disable-engine",
		ConfigPath:        types.EnginesDisabledKey,
		DefaultValue:      types.Default.Engines.Disabled,
		Description:       "Engine types to disable",
		Deprecated:        true,
		DeprecatedMessage: makeDeprecationMessage(types.EnginesDisabledKey),
	},
	{
		FlagName:          "disabled-publisher",
		ConfigPath:        types.PublishersDisabledKey,
		DefaultValue:      types.Default.Publishers.Disabled,
		Description:       "Publisher types to disable",
		Deprecated:        true,
		DeprecatedMessage: makeDeprecationMessage(types.PublishersDisabledKey),
	},
	{
		FlagName:          "disable-storage",
		ConfigPath:        types.InputSourcesDisabledKey,
		DefaultValue:      types.Default.InputSources.Disabled,
		Description:       "Storage types to disable",
		Deprecated:        true,
		DeprecatedMessage: makeDeprecationMessage(types.InputSourcesDisabledKey),
	},
}
