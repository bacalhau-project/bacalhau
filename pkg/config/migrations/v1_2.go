package migrations

import "github.com/bacalhau-project/bacalhau/pkg/config/types"

var (
	oldSwarmPeers = []string{
		"/ip4/35.245.115.191/tcp/4001/p2p/12D3KooWE4wfAknWtY9mQ4eAA8zrFGeZa7X2Kh4nBP2tZgDSt7Rh",
		"/ip4/35.245.115.191/udp/4001/quic/p2p/12D3KooWE4wfAknWtY9mQ4eAA8zrFGeZa7X2Kh4nBP2tZgDSt7Rh",
		"/ip4/35.245.61.251/tcp/4001/p2p/12D3KooWD8zeukHTMyuPtQBoUUPqtEnaA7NwFXWcVywUJtCVPske",
		"/ip4/35.245.61.251/udp/4001/quic/p2p/12D3KooWD8zeukHTMyuPtQBoUUPqtEnaA7NwFXWcVywUJtCVPske",
		"/ip4/35.245.251.239/tcp/4001/p2p/12D3KooWAg1YdehZxcZhetcgA6KP8TLGX6Fq4h9PUswnUWoStVNc",
		"/ip4/35.245.251.239/udp/4001/quic/p2p/12D3KooWAg1YdehZxcZhetcgA6KP8TLGX6Fq4h9PUswnUWoStVNc",
		"/ip4/34.150.153.87/tcp/4001/p2p/12D3KooWGE4R98vokeLsRVdTv8D6jhMnifo81mm7NMRV8WJPNVHb",
		"/ip4/34.150.153.87/udp/4001/quic/p2p/12D3KooWGE4R98vokeLsRVdTv8D6jhMnifo81mm7NMRV8WJPNVHb",
		"/ip4/34.91.247.176/tcp/4001/p2p/12D3KooWSNKPM5PBchoqn774bpQ4j4QbL3VoyX6mH6vTyWXqE3kH",
		"/ip4/34.91.247.176/udp/4001/quic/p2p/12D3KooWSNKPM5PBchoqn774bpQ4j4QbL3VoyX6mH6vTyWXqE3kH",
	}

	oldBootstrapPeers = []string{
		"/ip4/35.245.115.191/tcp/1235/p2p/QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
		"/ip4/35.245.61.251/tcp/1235/p2p/QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
		"/ip4/35.245.251.239/tcp/1235/p2p/QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3",
	}
)

//noling:gochecknoinits
func init() {
	// this migration removes any IPFS swarm peers or Bootstrap peers that are incorrect from the v1.0.4 upgrade.
	// if no incorrect values are present they are left as is.
	set.Register(1, func(current types.BacalhauConfig) (types.BacalhauConfig, error) {
		migrated := current
		if haveSameElements(oldSwarmPeers, migrated.Node.IPFS.SwarmAddresses) {
			migrated.Node.IPFS.SwarmAddresses = []string{}
		}
		if haveSameElements(oldBootstrapPeers, migrated.Node.BootstrapAddresses) {
			migrated.Node.BootstrapAddresses = []string{}
		}
		return migrated, nil
	})
}

// haveSameElements returns true if arr1 and arr2 have the same elements, false otherwise.
func haveSameElements(arr1, arr2 []string) bool {
	if len(arr1) != len(arr2) {
		return false
	}

	elementCount := make(map[string]int)

	for _, item := range arr1 {
		elementCount[item]++
	}

	for _, item := range arr2 {
		if count, exists := elementCount[item]; !exists || count == 0 {
			return false
		}
		elementCount[item]--
	}

	return true
}
