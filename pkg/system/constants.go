package system

// EnvironmentData captures data for a particular environment.
type EnvironmentData struct {
	// APIHost is the hostname of an environment's public API servers.
	APIHost string

	// APIPort is the port that an environment serves the public API on.
	APIPort int

	// Bootstrap lists the bacalhau addresses for bootstrapping new local nodes.
	BootstrapAddresses []string

	// IPFSSwarmAddresses lists the swarm addresses of an environment's IPFS
	// nodes, for bootstrapping new local nodes.
	IPFSSwarmAddresses []string
}

// Envs is a list of environment data for various environments:
var Envs = map[Environment]EnvironmentData{
	EnvironmentProd: {
		APIPort: 1234,
		APIHost: "bootstrap.production.bacalhau.org",
		BootstrapAddresses: []string{
			"/ip4/35.245.115.191/tcp/1235/p2p/QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
			"/ip4/35.245.61.251/tcp/1235/p2p/QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
			"/ip4/35.245.251.239/tcp/1235/p2p/QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3",
		},
		IPFSSwarmAddresses: []string{
			"/ip4/35.245.115.191/tcp/1235/p2p/QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
			"/ip4/35.245.61.251/tcp/1235/p2p/QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
			"/ip4/35.245.251.239/tcp/1235/p2p/QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3",
		},
	},
	EnvironmentDev: {
		APIPort: 1234,
		APIHost: "bootstrap.development.bacalhau.org",
		BootstrapAddresses: []string{
			"/ip4/34.88.147.110/tcp/1235/p2p/QmNXczFhX8oLEeuGThGowkcJDJUnX4HqoYQ2uaYhuCNSxD",
			"/ip4/34.88.135.65/tcp/1235/p2p/QmfRDVYnEcPassyJFGQw8Wt4t9QuA843uuKPVNEVNm4Smo",
		},
		IPFSSwarmAddresses: []string{
			"/ip4/34.88.147.110/tcp/1235/p2p/QmNXczFhX8oLEeuGThGowkcJDJUnX4HqoYQ2uaYhuCNSxD",
			"/ip4/34.88.135.65/tcp/1235/p2p/QmfRDVYnEcPassyJFGQw8Wt4t9QuA843uuKPVNEVNm4Smo",
		},
	},
	EnvironmentStaging: {
		APIPort: 1234,
		APIHost: "bootstrap.staging.bacalhau.org",
		BootstrapAddresses: []string{
			"/ip4/35.199.72.224/tcp/1235/p2p/QmP6RVpStuEoShqTTTiS2e3PYazcd54sj2RaZTeJP9VCeh",
			"/ip4/35.198.1.209/tcp/1235/p2p/QmU7NmyuztsYPeLxrw3B3p97bZfJD5PRL9igvhDepfhsGY",
			"/ip4/35.247.208.185/tcp/1235/p2p/QmWhWSQLxuGARV2g8oXt2v5HVfwYWqRCDAuGVz6qDnS6kX",
		},
		IPFSSwarmAddresses: []string{
			"/ip4/35.199.72.224/tcp/1235/p2p/QmP6RVpStuEoShqTTTiS2e3PYazcd54sj2RaZTeJP9VCeh",
			"/ip4/35.198.1.209/tcp/1235/p2p/QmU7NmyuztsYPeLxrw3B3p97bZfJD5PRL9igvhDepfhsGY",
			"/ip4/35.247.208.185/tcp/1235/p2p/QmWhWSQLxuGARV2g8oXt2v5HVfwYWqRCDAuGVz6qDnS6kX",
		},
	},
}
