package configflags

import "github.com/bacalhau-project/bacalhau/pkg/config/types"

var IPFSFlags = []Definition{
	{
		FlagName:             "ipfs-swarm-addrs",
		ConfigPath:           types.NodeIPFSSwarmAddresses,
		DefaultValue:         Default.Node.IPFS.SwarmAddresses,
		Description:          "IPFS multiaddress to connect the in-process IPFS node to - cannot be used with --ipfs-connect.",
		EnvironmentVariables: []string{"BACALHAU_IPFS_SWARM_ADDRESSES"},
	},
	{
		FlagName:             "ipfs-swarm-key",
		ConfigPath:           types.NodeIPFSSwarmKeyPath,
		DefaultValue:         Default.Node.IPFS.SwarmKeyPath,
		Description:          "Optional IPFS swarm key required to connect to a private IPFS swarm",
		EnvironmentVariables: []string{"BACALHAU_IPFS_SWARM_KEY"},
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
	{
		FlagName:             "ipfs-serve-path",
		ConfigPath:           types.NodeIPFSServePath,
		DefaultValue:         Default.Node.IPFS.ServePath,
		Description:          "path local Ipfs node will persist data to",
		EnvironmentVariables: []string{"BACALHAU_SERVE_IPFS_PATH"},
	},
	{
		FlagName:     "ipfs-profile",
		ConfigPath:   types.NodeIPFSProfile,
		DefaultValue: Default.Node.IPFS.Profile,
		Description:  "profile for internal IPFS node",
	},
	{
		FlagName:     "ipfs-swarm-listen-addresses",
		ConfigPath:   types.NodeIPFSSwarmListenAddresses,
		DefaultValue: Default.Node.IPFS.SwarmListenAddresses,
		Description:  "addresses the internal IPFS node will listen on for swarm connections",
	},
	{
		FlagName:     "ipfs-gateway-listen-addresses",
		ConfigPath:   types.NodeIPFSGatewayListenAddresses,
		DefaultValue: Default.Node.IPFS.GatewayListenAddresses,
		Description:  "addresses the internal IPFS node will listen on for gateway connections",
	},
	{
		FlagName:     "ipfs-api-listen-addresses",
		ConfigPath:   types.NodeIPFSAPIListenAddresses,
		DefaultValue: Default.Node.IPFS.APIListenAddresses,
		Description:  "addresses the internal IPFS node will listen on for API connections",
	},
}
