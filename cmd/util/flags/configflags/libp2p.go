package configflags

import "github.com/bacalhau-project/bacalhau/pkg/config/types"

var Libp2pFlags = []Definition{
	{
		FlagName:     "peer",
		ConfigPath:   types.NodeLibp2pPeerConnect,
		DefaultValue: Default.Node.Libp2p.PeerConnect,
		Description: `A comma-separated list of libp2p multiaddress to connect to. ` +
			`Use "none" to avoid connecting to any peer, ` +
			`"env" to connect to the default peer list of your active environment (see BACALHAU_ENVIRONMENT env var).`,
		Deprecated: true,
		DeprecatedMessage: "The libp2p transport will be deprecated in a future version in favour of using " +
			"--orchestrators to specify a requester node to connect to.",
	},
	{
		FlagName:     "swarm-port",
		ConfigPath:   types.NodeLibp2pSwarmPort,
		DefaultValue: Default.Node.Libp2p.SwarmPort,
		Description:  `The port to listen on for swarm connections.`,
		Deprecated:   true,
		DeprecatedMessage: "The libp2p transport will be deprecated in a future version in favour of using " +
			"--orchestrators to specify a requester node to connect to.",
	},
}
