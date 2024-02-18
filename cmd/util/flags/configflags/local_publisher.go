package configflags

import "github.com/bacalhau-project/bacalhau/pkg/config/types"

var LocalPublisherFlags = []Definition{
	{
		FlagName:     "local-publisher-address",
		DefaultValue: Default.Node.Compute.LocalPublisher.Address,
		ConfigPath:   types.NodeComputeLocalPublisherAddress,
		Description:  `The address for the local publisher's server to bind to`,
	},
	{
		FlagName:     "local-publisher-port",
		DefaultValue: Default.Node.Compute.LocalPublisher.Port,
		ConfigPath:   types.NodeComputeLocalPublisherPort,
		Description:  `The port for the local publisher's server to bind to (default: 6001)`,
	},
	{
		FlagName:     "local-publisher-directory",
		DefaultValue: Default.Node.Compute.LocalPublisher.Directory,
		ConfigPath:   types.NodeComputeLocalPublisherDirectory,
		Description:  `The directory where the local publisher will store content`,
	},
}
