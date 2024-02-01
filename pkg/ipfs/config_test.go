//go:build unit || !integration

package ipfs

import (
	"testing"

	"github.com/ipfs/kubo/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

func TestSetSwarmListenAddresses(t *testing.T) {
	expected := []string{
		"/ip4/10.0.0.1/tcp/1234",
		"/ip4/10.0.0.2/tcp/1234",
	}

	ipfscfg, err := buildIPFSConfig(types.IpfsConfig{
		PrivateInternal:      false,
		SwarmListenAddresses: expected,
	})
	require.NoError(t, err)

	assert.ElementsMatch(t, ipfscfg.Addresses.Swarm, expected)
	assert.ElementsMatch(t, ipfscfg.Addresses.Gateway, []string{"/ip4/0.0.0.0/tcp/0", "/ip6/::1/tcp/0"})
	assert.ElementsMatch(t, ipfscfg.Addresses.API, []string{"/ip4/0.0.0.0/tcp/0", "/ip6/::1/tcp/0"})
}

func TestSetGatewayListenAddresses(t *testing.T) {
	expected := []string{
		"/ip4/10.0.0.1/tcp/1234",
		"/ip4/10.0.0.2/tcp/1234",
	}

	ipfscfg, err := buildIPFSConfig(types.IpfsConfig{
		PrivateInternal:        false,
		GatewayListenAddresses: expected,
	})
	require.NoError(t, err)

	assert.ElementsMatch(t, ipfscfg.Addresses.Gateway, expected)
	assert.ElementsMatch(t, ipfscfg.Addresses.Swarm, []string{"/ip4/0.0.0.0/tcp/0", "/ip6/::1/tcp/0"})
	assert.ElementsMatch(t, ipfscfg.Addresses.API, []string{"/ip4/0.0.0.0/tcp/0", "/ip6/::1/tcp/0"})
}

func TestSetAPIListenAddresses(t *testing.T) {
	expected := []string{
		"/ip4/10.0.0.1/tcp/1234",
		"/ip4/10.0.0.2/tcp/1234",
	}

	ipfscfg, err := buildIPFSConfig(types.IpfsConfig{
		PrivateInternal:    false,
		APIListenAddresses: expected,
	})
	require.NoError(t, err)

	assert.ElementsMatch(t, ipfscfg.Addresses.API, expected)
	assert.ElementsMatch(t, ipfscfg.Addresses.Swarm, []string{"/ip4/0.0.0.0/tcp/0", "/ip6/::1/tcp/0"})
	assert.ElementsMatch(t, ipfscfg.Addresses.Gateway, []string{"/ip4/0.0.0.0/tcp/0", "/ip6/::1/tcp/0"})
}

func TestPrivateConfig(t *testing.T) {
	ipfscfg, err := buildIPFSConfig(types.IpfsConfig{
		PrivateInternal: true,
	})
	require.NoError(t, err)

	assert.ElementsMatch(t, ipfscfg.Addresses.Swarm, []string{"/ip4/127.0.0.1/tcp/0"})
	assert.ElementsMatch(t, ipfscfg.Addresses.API, []string{"/ip4/0.0.0.0/tcp/0", "/ip6/::1/tcp/0"})
	assert.ElementsMatch(t, ipfscfg.Addresses.Gateway, []string{"/ip4/0.0.0.0/tcp/0", "/ip6/::1/tcp/0"})
	assert.Equal(t, config.AutoNATServiceDisabled, ipfscfg.AutoNAT.ServiceMode)
	assert.Equal(t, config.False, ipfscfg.Swarm.EnableHolePunching)
	assert.Equal(t, config.False, ipfscfg.Swarm.RelayClient.Enabled)
	assert.Equal(t, config.False, ipfscfg.Swarm.RelayService.Enabled)
	assert.Equal(t, config.False, ipfscfg.Swarm.Transports.Network.Relay)
}

func TestWithPublicSwarm(t *testing.T) {
	expected := []string{"/ip4/127.0.0.1/tcp/34441/p2p/QmRgRvyNXV79vBMKv8tJVwwScAjeTn2F7QU3W4xSZTmEjq"}
	ipfscfg, err := buildIPFSConfig(types.IpfsConfig{
		SwarmAddresses: expected,
	})
	require.NoError(t, err)

	swarmPeers, err := ParsePeersString(expected)
	require.NoError(t, err)
	assert.Equal(t, ipfscfg.Peering, config.Peering{Peers: swarmPeers})
}

func TestWithPrivateSwarm(t *testing.T) {
	expected := []string{"/ip4/127.0.0.1/tcp/34441/p2p/QmRgRvyNXV79vBMKv8tJVwwScAjeTn2F7QU3W4xSZTmEjq"}
	ipfscfg, err := buildIPFSConfig(types.IpfsConfig{
		SwarmAddresses: expected,
		SwarmKeyPath:   "/some/path/to/key",
	})
	require.NoError(t, err)

	swarmPeers, err := ParsePeersString(expected)
	require.NoError(t, err)
	assert.Equal(t, ipfscfg.Peering, config.Peering{Peers: swarmPeers})
	assert.Equal(t, ipfscfg.Bootstrap, expected)
}
