package configflags

import (
	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
)

var LocalPublisherFlags = []Definition{
	{
		FlagName:     "local-publisher-address",
		DefaultValue: types2.Default.Publishers.Local.Address,
		ConfigPath:   types2.PublishersLocalAddressKey,
		Description:  `The address for the local publisher's server to bind to`,
	},
	{
		FlagName:     "local-publisher-port",
		DefaultValue: types2.PublishersLocalPortKey,
		ConfigPath:   "Publisher.Local.Port",
		Description:  `The port for the local publisher's server to bind to (default: 6001)`,
	},
	{
		FlagName:     "local-publisher-directory",
		DefaultValue: types2.PublishersLocalDirectoryKey,
		ConfigPath:   "Publisher.Local.Directory",
		Description:  `The directory where the local publisher will store content`,
	},
}
