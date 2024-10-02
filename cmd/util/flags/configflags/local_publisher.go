package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

var LocalPublisherFlags = []Definition{
	{
		FlagName:             "local-publisher-address",
		DefaultValue:         config.Default.Publishers.Types.Local.Address,
		ConfigPath:           types.PublishersTypesLocalAddressKey,
		EnvironmentVariables: []string{"BACALHAU_NODE_COMPUTE_LOCALPUBLISHER_ADDRESS"},
		Description:          `The address for the local publisher's server to bind to.`,
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.PublishersTypesLocalAddressKey),
	},
	{
		FlagName:             "local-publisher-port",
		DefaultValue:         config.Default.Publishers.Types.Local.Port,
		ConfigPath:           types.PublishersTypesLocalPortKey,
		Description:          `The port for the local publisher's server to bind to.`,
		EnvironmentVariables: []string{"BACALHAU_NODE_COMPUTE_LOCALPUBLISHER_PORT"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.PublishersTypesLocalPortKey),
	},
	{
		FlagName:          "local-publisher-directory",
		Deprecated:        true,
		DefaultValue:      "",
		DeprecatedMessage: FeatureDeprecatedMessage,
	},
}
