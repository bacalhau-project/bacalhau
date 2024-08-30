package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

var LocalPublisherFlags = []Definition{
	{
		FlagName:     "local-publisher-address",
		DefaultValue: types.Default.Publishers.Local.Address,
		ConfigPath:   types.PublishersLocalAddressKey,
		Description:  `The address for the local publisher's server to bind to`,
	},
	{
		FlagName:     "local-publisher-port",
		DefaultValue: types.PublishersLocalPortKey,
		ConfigPath:   "Publisher.Local.Port",
		Description:  `The port for the local publisher's server to bind to (default: 6001)`,
	},
	{
		FlagName:     "local-publisher-directory",
		DefaultValue: types.PublishersLocalDirectoryKey,
		ConfigPath:   "Publisher.Local.Directory",
		Description:  `The directory where the local publisher will store content`,
	},
}
