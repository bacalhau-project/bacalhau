package system

// EnvironmentData captures data for a particular environment.
type EnvironmentData struct {
	// APIHost is the hostname of an environment's public API servers.
	APIHost string

	// APIPort is the port that an environment serves the public API on.
	APIPort uint16

	// Bootstrap lists the bacalhau addresses for bootstrapping new local nodes.
	BootstrapAddresses []string

	// IPFSSwarmAddresses lists the swarm addresses of an environment's IPFS
	// nodes, for bootstrapping new local nodes.
	IPFSSwarmAddresses []string
}

// Envs is a list of environment data for various environments:
// Deprecated: stop using this, and use the config file.
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
			"/ip4/34.86.177.175/tcp/1235/p2p/QmfYBQ3HouX9zKcANNXbgJnpyLpTYS9nKBANw6RUQKZffu",
			"/ip4/35.245.221.171/tcp/1235/p2p/QmNjEQByyK8GiMTvnZqGyURuwXDCtzp9X6gJRKkpWfai7S",
		},
		IPFSSwarmAddresses: []string{
			"/ip4/34.86.177.175/tcp/4001/p2p/12D3KooWMSdbPzUf8WWkEcjxpCzkUfToasP9wRjFHy2iCZ6iiZdV",
			"/ip4/34.86.177.175/udp/4001/quic/p2p/12D3KooWMSdbPzUf8WWkEcjxpCzkUfToasP9wRjFHy2iCZ6iiZdV",
			"/ip4/35.245.221.171/tcp/4001/p2p/12D3KooWRBYMhTF6MNh6eN84xcZtg6EX2wJguqEtRTNq4C7aytbu",
			"/ip4/35.245.221.171/udp/4001/quic/p2p/12D3KooWRBYMhTF6MNh6eN84xcZtg6EX2wJguqEtRTNq4C7aytbu",
		},
	},
	EnvironmentStaging: {
		APIPort: 1234,
		APIHost: "bootstrap.staging.bacalhau.org",
		BootstrapAddresses: []string{
			"/ip4/34.85.228.65/tcp/1235/p2p/QmafZ9oCXCJZX9Wt1nhrGS9FVVq41qhcBRSNWCkVhz3Nvv",
			"/ip4/34.86.73.105/tcp/1235/p2p/QmVHCeiLzhFJPCyCj5S1RTAk1vBEvxd8r5A6E4HyJGQtbJ",
			"/ip4/34.150.138.100/tcp/1235/p2p/QmRr9qPTe4mU7aS9faKnWgvn1NtXt36FT8YUULRPCn2f3K",
		},
		IPFSSwarmAddresses: []string{
			"/ip4/34.85.228.65/tcp/4001/p2p/12D3KooWCWSTjjWh7SVoVv24W47z3T1Ly1tgnwZ56CCqCku5e4dS",
			"/ip4/34.85.228.65/udp/4001/quic/p2p/12D3KooWCWSTjjWh7SVoVv24W47z3T1Ly1tgnwZ56CCqCku5e4dS",
			"/ip4/34.86.73.105/tcp/4001/p2p/12D3KooWQuhW3LSpvhea25Zed47Z7fD5Cq2nw1xmapQ2tAUJ3q4F",
			"/ip4/34.86.73.105/udp/4001/quic/p2p/12D3KooWQuhW3LSpvhea25Zed47Z7fD5Cq2nw1xmapQ2tAUJ3q4F",
			"/ip4/34.150.138.100/tcp/4001/p2p/12D3KooWQm1T8EN8fMBz7rLviHxTGdRnohZ9nDPGbW4bfi78ckVT",
			"/ip4/34.150.138.100/udp/4001/quic/p2p/12D3KooWQm1T8EN8fMBz7rLviHxTGdRnohZ9nDPGbW4bfi78ckVT",
			"/ip4/35.245.247.85/tcp/4001/p2p/12D3KooWEztGEJtqtzy7th2d7cTw2iR4CQCPHFUYvj66rhh9Cf7h",
			"/ip4/35.245.247.85/udp/4001/quic/p2p/12D3KooWEztGEJtqtzy7th2d7cTw2iR4CQCPHFUYvj66rhh9Cf7h",
		},
	},
	EnvironmentTest: {
		APIPort: 9999,
		APIHost: "test",
		BootstrapAddresses: []string{
			"/ip4/0.0.0.0/tcp/1235/p2p/QmafZ9oCXCJZX9Wt1nhrGS9FVVq41qhcBRSNWCkVhz3Nvv",
		},
		IPFSSwarmAddresses: []string{
			"/ip4/0.0.0.0/tcp/1235/p2p/QmafZ9oCXCJZX9Wt1nhrGS9FVVq41qhcBRSNWCkVhz3Nvv",
		},
	},
}
