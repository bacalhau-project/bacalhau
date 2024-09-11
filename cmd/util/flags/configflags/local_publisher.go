package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

var LocalPublisherFlags = []Definition{
	{
		FlagName:          "local-publisher-address",
		DefaultValue:      types.Default.Publishers.Types.Local.Address,
		ConfigPath:        types.PublishersTypesLocalAddressKey,
		Description:       `The address for the local publisher's server to bind to.`,
		Deprecated:        true,
		DeprecatedMessage: makeDeprecationMessage(types.PublishersTypesLocalAddressKey),
	},
	{
		FlagName:          "local-publisher-port",
		DefaultValue:      types.Default.Publishers.Types.Local.Port,
		ConfigPath:        types.PublishersTypesLocalPortKey,
		Description:       `The port for the local publisher's server to bind to.`,
		Deprecated:        true,
		DeprecatedMessage: makeDeprecationMessage(types.PublishersTypesLocalPortKey),
	},
	{
		FlagName:          "local-publisher-directory",
		DefaultValue:      types.Default.Publishers.Types.Local.Directory,
		ConfigPath:        types.PublishersTypesLocalDirectoryKey,
		Description:       `The directory where the local publisher will store content.`,
		Deprecated:        true,
		DeprecatedMessage: makeDeprecationMessage(types.PublishersTypesLocalDirectoryKey),
	},
}
