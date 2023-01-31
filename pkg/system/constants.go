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
			"/ip4/34.125.80.77/tcp/1235/p2p/QmRbFXwNRamB8z3SXvSg6KPCHGDAyYKjggARU1KBTxWeMz",
			"/ip4/34.125.130.185/tcp/1235/p2p/QmTRsYgLck8RHedBdZijjA8jPfkSKjmUwVu2k9iUXJ7SLp",
			"/ip4/34.125.197.225/tcp/1235/p2p/QmS8wQaZqyRXP83rZQtNsmLzgU9ZJYxAVVTDETFdEFZXGA",
		},
		IPFSSwarmAddresses: []string{
			"/ip4/34.125.80.77/tcp/1235/p2p/QmRbFXwNRamB8z3SXvSg6KPCHGDAyYKjggARU1KBTxWeMz",
			"/ip4/34.125.130.185/tcp/1235/p2p/QmTRsYgLck8RHedBdZijjA8jPfkSKjmUwVu2k9iUXJ7SLp",
			"/ip4/34.125.197.225/tcp/1235/p2p/QmS8wQaZqyRXP83rZQtNsmLzgU9ZJYxAVVTDETFdEFZXGA",
		},
	},
}
