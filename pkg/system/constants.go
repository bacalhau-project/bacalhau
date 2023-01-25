package system

// EnvironmentType is a type of environment (prod, staging, dev etcetera).
type EnvironmentType int

const (
	Development EnvironmentType = iota
	Staging
	Production
)

// EnvironmentData captures data for a particular environment.
type EnvironmentData struct {
	// APIHost is the hostname of an environment's public API servers.
	APIHost string

	// APIPort is the port that an environment serves the public API on.
	APIPort int

	// IPFSSwarmAddresses lists the swarm addresses of an environment's IPFS
	// nodes, for bootstrapping new local nodes.
	IPFSSwarmAddresses []string
}

// Envs is a list of environment data for various environments:
var Envs = map[EnvironmentType]EnvironmentData{
	Production: {
		APIPort: 1234,
		APIHost: "bootstrap.production.bacalhau.org",
		IPFSSwarmAddresses: []string{
			"/ip4/35.245.115.191/tcp/1235/p2p/QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
			"/ip4/35.245.61.251/tcp/1235/p2p/QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
			"/ip4/35.245.251.239/tcp/1235/p2p/QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3",
		},
	},

	// TODO: fill these in
	Development: {},
	Staging: {
		APIPort: 1234,
		APIHost: "bootstrap.staging.bacalhau.org",
		IPFSSwarmAddresses: []string{
			"/ip4/35.199.72.224/tcp/1235/p2p/QmP6RVpStuEoShqTTTiS2e3PYazcd54sj2RaZTeJP9VCeh",
			"/ip4/35.198.1.209/tcp/1235/p2p/QmU7NmyuztsYPeLxrw3B3p97bZfJD5PRL9igvhDepfhsGY",
			"/ip4/35.247.208.185/tcp/1235/p2p/QmWhWSQLxuGARV2g8oXt2v5HVfwYWqRCDAuGVz6qDnS6kX",
		},
	},
}
