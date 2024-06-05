package migrations

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

var (
	oldSwarmPeers = []string{
		"/ip4/35.245.115.191/tcp/4001/p2p/12D3KooWE4wfAknWtY9mQ4eAA8zrFGeZa7X2Kh4nBP2tZgDSt7Rh",
		"/ip4/35.245.61.251/tcp/4001/p2p/12D3KooWD8zeukHTMyuPtQBoUUPqtEnaA7NwFXWcVywUJtCVPske",
		"/ip4/35.245.251.239/tcp/4001/p2p/12D3KooWAg1YdehZxcZhetcgA6KP8TLGX6Fq4h9PUswnUWoStVNc",
		"/ip4/34.150.153.87/tcp/4001/p2p/12D3KooWGE4R98vokeLsRVdTv8D6jhMnifo81mm7NMRV8WJPNVHb",
		"/ip4/34.91.247.176/tcp/4001/p2p/12D3KooWSNKPM5PBchoqn774bpQ4j4QbL3VoyX6mH6vTyWXqE3kH",
	}

	oldBootstrapPeers = []string{
		"/ip4/35.245.115.191/tcp/1235/p2p/QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
		"/ip4/35.245.61.251/tcp/1235/p2p/QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
		"/ip4/35.245.251.239/tcp/1235/p2p/QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3",
	}
)

var V1Migration = repo.NewMigration(
	repo.RepoVersion1,
	repo.RepoVersion2,
	func(r repo.FsRepo) error {
		configExist, err := configExists(r)
		if err != nil {
			return err
		}
		if !configExist {
			return nil
		}
		v, cfg, err := readConfig(r)
		if err != nil {
			return err
		}

		// this migration removes any IPFS swarm peers or Bootstrap peers that are incorrect from the v1.0.4 upgrade.
		// if no incorrect values are present they are left as is.
		doWrite := false
		if haveSameElements(oldSwarmPeers, cfg.Node.IPFS.SwarmAddresses) {
			v.Set(types.NodeIPFSSwarmAddresses, []string{})
			doWrite = true
		}
		if haveSameElements(oldBootstrapPeers, cfg.Node.BootstrapAddresses) {
			v.Set(types.NodeBootstrapAddresses, []string{})
			doWrite = true
		}

		if doWrite {
			return v.WriteConfig()
		}
		return nil
	})
