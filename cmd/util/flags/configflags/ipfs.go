package configflags

import "github.com/bacalhau-project/bacalhau/pkg/config/types"

var IPFSFlags = []Definition{
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
		FlagName:             "ipfs-serve-path",
		ConfigPath:           types.NodeIPFSServePath,
		DefaultValue:         Default.Node.IPFS.ServePath,
		Description:          "path local Ipfs node will persist data to",
		EnvironmentVariables: []string{"BACALHAU_SERVE_IPFS_PATH"},
	},
}
