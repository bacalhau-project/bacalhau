package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

var NodeNameFlags = []Definition{
	{
		FlagName:             "name-provider",
		ConfigPath:           types.NameProviderKey,
		DefaultValue:         types.Default.NameProvider,
		Description:          `The name provider to use to generate the node name when the node initializes.`,
		EnvironmentVariables: []string{"BACALHAU_NODE_NAMEPROVIDER"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.NameProviderKey),
	},
}
