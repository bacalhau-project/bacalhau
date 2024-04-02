package configflags

import "github.com/bacalhau-project/bacalhau/pkg/config/types"

var DisabledFeatureFlags = []Definition{
	{
		FlagName:     "disable-engine",
		ConfigPath:   types.NodeDisabledFeaturesEngines,
		DefaultValue: Default.Node.DisabledFeatures.Engines,
		Description:  "Engine types to disable",
	},
	{
		FlagName:     "disabled-publisher",
		ConfigPath:   types.NodeDisabledFeaturesPublishers,
		DefaultValue: Default.Node.DisabledFeatures.Publishers,
		Description:  "Publisher types to disable",
	},
	{
		FlagName:     "disable-storage",
		ConfigPath:   types.NodeDisabledFeaturesStorages,
		DefaultValue: Default.Node.DisabledFeatures.Storages,
		Description:  "Storage types to disable",
	},
}
