package configflags

import (
	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
)

var NodeNameFlags = []Definition{
	{
		FlagName:     "name-provider",
		ConfigPath:   "NameProvider",
		DefaultValue: types2.Default.NameProvider,
		Description:  `The name provider to use to generate the node name when the node initializes.`,
	},
}
