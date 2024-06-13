package configflags

import "github.com/bacalhau-project/bacalhau/pkg/config/types"

var IPFSFlags = []Definition{
	{
		FlagName:     "ipfs-connect",
		ConfigPath:   types.NodeIPFSConnect,
		DefaultValue: Default.Node.IPFS.Connect,
		Description:  "The ipfs host multiaddress to connect to, otherwise an in-process IPFS node will be created if not set.",
	},
}
