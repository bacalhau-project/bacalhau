package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/cfgtypes"
)

var LocalPublisherFlags = []Definition{
	{
		FlagName:     "local-publisher-address",
		DefaultValue: cfgtypes.Default.Publishers.Local.Address,
		ConfigPath:   cfgtypes.PublishersLocalAddressKey,
		Description:  `The address for the local publisher's server to bind to`,
	},
	{
		FlagName:     "local-publisher-port",
		DefaultValue: cfgtypes.PublishersLocalPortKey,
		ConfigPath:   "Publisher.Local.Port",
		Description:  `The port for the local publisher's server to bind to (default: 6001)`,
	},
	{
		FlagName:     "local-publisher-directory",
		DefaultValue: cfgtypes.PublishersLocalDirectoryKey,
		ConfigPath:   "Publisher.Local.Directory",
		Description:  `The directory where the local publisher will store content`,
	},
}
