package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/cfgtypes"
)

var NodeNameFlags = []Definition{
	{
		FlagName:     "name-provider",
		ConfigPath:   cfgtypes.NameProviderKey,
		DefaultValue: cfgtypes.Default.NameProvider,
		Description:  `The name provider to use to generate the node name when the node initializes.`,
	},
}
