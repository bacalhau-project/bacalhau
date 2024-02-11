package configflags

import "github.com/bacalhau-project/bacalhau/pkg/config/types"

var NodeNameFlags = []Definition{
	{
		FlagName:     "name",
		ConfigPath:   types.NodeName,
		DefaultValue: "",
		Description:  `The name of the node. If not set, the node name will be generated automatically based on the chosen name provider.`,
	},
	{
		FlagName:     "name-provider",
		ConfigPath:   types.NodeNameProvider,
		DefaultValue: Default.Node.NameProvider,
		Description:  `The name provider to use to generate the node name, if no name is set.`,
	},
}
