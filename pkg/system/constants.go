package system

// EnvironmentData captures data for a particular environment.
type EnvironmentData struct {
	// APIHost is the hostname of an environment's public API servers.
	APIHost string

	// APIPort is the port that an environment serves the public API on.
	APIPort uint16

	// Bootstrap lists the bacalhau addresses for bootstrapping new local nodes.
	BootstrapAddresses []string
}

// Envs is a list of environment data for various environments:
// Deprecated: stop using this, and use the config file.
var Envs = map[Environment]EnvironmentData{
	EnvironmentProd: {
		APIPort: 1234,
		APIHost: "bootstrap.production.bacalhau.org",
		BootstrapAddresses: []string{
			"/ip4/35.245.161.250/tcp/1235/p2p/QmbxGSsM6saCTyKkiWSxhJCt6Fgj7M9cns1vzYtfDbB5Ws",
			"/ip4/34.86.254.26/tcp/1235/p2p/QmeXjeQDinxm7zRiEo8ekrJdbs7585BM6j7ZeLVFrA7GPe",
			"/ip4/35.245.215.155/tcp/1235/p2p/QmPLPUUjaVE3wQNSSkxmYoaBPHVAWdjBjDYmMkWvtMZxAf",
		},
	},
	EnvironmentDev: {
		APIPort: 1234,
		APIHost: "bootstrap.development.bacalhau.org",
		BootstrapAddresses: []string{
			"/ip4/34.86.177.175/tcp/1235/p2p/QmfYBQ3HouX9zKcANNXbgJnpyLpTYS9nKBANw6RUQKZffu",
			"/ip4/35.245.221.171/tcp/1235/p2p/QmNjEQByyK8GiMTvnZqGyURuwXDCtzp9X6gJRKkpWfai7S",
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
	},
	EnvironmentTest: {
		APIPort: 9999,
		APIHost: "test",
		BootstrapAddresses: []string{
			"/ip4/0.0.0.0/tcp/1235/p2p/QmafZ9oCXCJZX9Wt1nhrGS9FVVq41qhcBRSNWCkVhz3Nvv",
		},
	},
}
