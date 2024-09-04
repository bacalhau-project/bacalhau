package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

var LocalPublisherFlags = []Definition{
	{
		FlagName:          "local-publisher-address",
		DefaultValue:      types.Default.Publishers.Local.Address,
		ConfigPath:        types.PublishersLocalAddressKey,
		Description:       `The address for the local publisher's server to bind to.`,
		Deprecated:        true,
		DeprecatedMessage: makeDeprecationMessage(types.PublishersLocalAddressKey),
	},
	{
		FlagName:          "local-publisher-port",
		DefaultValue:      types.Default.Publishers.Local.Port,
		ConfigPath:        types.PublishersLocalPortKey,
		Description:       `The port for the local publisher's server to bind to.`,
		Deprecated:        true,
		DeprecatedMessage: makeDeprecationMessage(types.PublishersLocalPortKey),
	},
	{
		FlagName:          "local-publisher-directory",
		DefaultValue:      types.Default.Publishers.Local.Directory,
		ConfigPath:        types.PublishersLocalDirectoryKey,
		Description:       `The directory where the local publisher will store content.`,
		Deprecated:        true,
		DeprecatedMessage: makeDeprecationMessage(types.PublishersLocalDirectoryKey),
	},
}
