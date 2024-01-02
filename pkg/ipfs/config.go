package ipfs

import (
	"fmt"
	"io"

	"github.com/ipfs/kubo/config"

	bac_config "github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

func buildIPFSConfig(cfg types.IpfsConfig) (*config.Config, error) {
	// NB(forrest): these empty checks are here because some tests pass
	// partial configs to these methods, and we cannot depend on the config being
	// fully populated with default values.
	profileName := cfg.Profile
	if profileName == "" {
		profileName = "flatfs"
	}

	swarmListenAddresses := cfg.SwarmListenAddresses
	if len(swarmListenAddresses) == 0 {
		swarmListenAddresses = []string{"/ip4/0.0.0.0/tcp/0", "/ip6/::1/tcp/0"}
	}

	gatewayListenAddresses := cfg.GatewayListenAddresses
	if len(gatewayListenAddresses) == 0 {
		gatewayListenAddresses = []string{"/ip4/0.0.0.0/tcp/0", "/ip6/::1/tcp/0"}
	}

	apiListenAddresses := cfg.APIListenAddresses
	if len(apiListenAddresses) == 0 {
		apiListenAddresses = []string{"/ip4/0.0.0.0/tcp/0", "/ip6/::1/tcp/0"}
	}

	profile := config.Profiles[profileName]
	transformers := []config.Transformer{
		withSwarmListenAddresses(swarmListenAddresses...),
		withGatewayListenAddresses(gatewayListenAddresses...),
		withAPIListenAddresses(apiListenAddresses...),
	}

	// If we're in local mode, then we need to manually change the config to
	// serve an IPFS swarm client on some local port. Else, make sure we are
	// only serving the API on a local connection
	if cfg.PrivateInternal {
		profile = config.Profiles["test"]
		transformers = append(transformers,
			// disable autonat, UPnP, hole-punching and relays
			withLocalOnly(),
			// only listen for swarm connections on local endpoint.
			withSwarmListenAddresses("/ip4/127.0.0.1/tcp/0"),
		)
	}
	if len(cfg.SwarmAddresses) > 0 {
		privateSwarm := cfg.SwarmKeyPath != ""
		transformers = append(transformers,
			withSwarm(cfg.GetSwarmAddresses(), privateSwarm))
	}

	ipfsConfig, err := config.Init(io.Discard, defaultKeypairSize)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize config: %w", err)
	}

	if err := profile.Transform(ipfsConfig); err != nil {
		return nil, err
	}

	for _, transformer := range transformers {
		if err := transformer(ipfsConfig); err != nil {
			return nil, err
		}
	}
	return ipfsConfig, nil
}

// withLocalOnly switches off networking services that might connect to public nodes
func withLocalOnly() config.Transformer {
	return func(c *config.Config) error {
		c.AutoNAT.ServiceMode = config.AutoNATServiceDisabled
		c.Swarm.EnableHolePunching = config.False
		c.Swarm.RelayClient.Enabled = config.False
		c.Swarm.RelayService.Enabled = config.False
		c.Swarm.Transports.Network.Relay = config.False
		return nil
	}
}

func withAPIListenAddresses(addrs ...string) config.Transformer {
	return func(c *config.Config) error {
		c.Addresses.API = addrs
		return nil
	}
}

func withGatewayListenAddresses(addrs ...string) config.Transformer {
	return func(c *config.Config) error {
		c.Addresses.Gateway = addrs
		return nil
	}
}

func withSwarmListenAddresses(addrs ...string) config.Transformer {
	// TODO(forrest): deprecate this someday.
	preferredAddr := bac_config.PreferredAddress()
	if preferredAddr != "" {
		return func(c *config.Config) error {
			c.Addresses.Swarm = []string{fmt.Sprintf("/ip4/%s/tcp/0", preferredAddr)}
			return nil
		}
	}
	return func(c *config.Config) error {
		c.Addresses.Swarm = addrs
		return nil
	}
}

// withSwarm will cause IPFS to continuously connect to the swarm.
// If the swarm is private, don't bootstrap with public nodes, only with swarm nodes.
func withSwarm(addrs []string, private bool) config.Transformer {
	return func(c *config.Config) error {
		// establish peering with the passed nodes. This is different than
		// bootstrapping or manually connecting to peers, and kubo will
		// create sticky connections with these nodes and reconnect if the
		// connection is lost
		// https://github.com/ipfs/kubo/blob/master/docs/config.md#peering
		swarmPeers, err := ParsePeersString(addrs)
		if err != nil {
			return fmt.Errorf("failed to parse peer addresses: %w", err)
		}
		c.Peering = config.Peering{Peers: swarmPeers}
		if private {
			c.Bootstrap = addrs
		}
		return nil
	}
}
