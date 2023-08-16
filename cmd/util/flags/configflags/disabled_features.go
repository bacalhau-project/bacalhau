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
		FlagName:     types.NodeDisabledFeaturesPublishers,
		ConfigPath:   "Node.DisabledFeature.Publishers",
		DefaultValue: Default.Node.DisabledFeatures.Publishers,
		Description:  "Engine types to disable",
	},
	{
		FlagName:     "disable-storage",
		ConfigPath:   types.NodeDisabledFeaturesStorages,
		DefaultValue: Default.Node.DisabledFeatures.Storages,
		Description:  "Engine types to disable",
	},
}
