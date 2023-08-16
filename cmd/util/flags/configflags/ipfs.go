package configflags

import "github.com/bacalhau-project/bacalhau/pkg/config/types"

var IPFSFlags = []Definition{
	{
		FlagName:     "ipfs-swarm-addr",
		ConfigPath:   types.NodeIPFSSwarmAddresses,
		DefaultValue: Default.Node.IPFS.SwarmAddresses,
		Description:  "IPFS multiaddress to connect the in-process IPFS node to - cannot be used with --ipfs-connect.",
	},
	{
		FlagName:     "ipfs-connect",
		ConfigPath:   types.NodeIPFSConnect,
		DefaultValue: Default.Node.IPFS.Connect,
		Description:  "The ipfs host multiaddress to connect to, otherwise an in-process IPFS node will be created if not set.",
	},
	{
		FlagName:     "private-internal-ipfs",
		ConfigPath:   types.NodeIPFSPrivateInternal,
		DefaultValue: Default.Node.IPFS.PrivateInternal,
		Description: "Whether the in-process IPFS node should auto-discover other nodes, including the public IPFS network - " +
			"cannot be used with --ipfs-connect. " +
			"Use \"--private-internal-ipfs=false\" to disable. " +
			"To persist a local Ipfs node, set BACALHAU_SERVE_IPFS_PATH to a valid path.",
	},
}
