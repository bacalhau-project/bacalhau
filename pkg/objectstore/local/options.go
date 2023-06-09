package local

type Option func(*LocalObjectConfig)

func WithDataFile(path string) Option {
	return func(c *LocalObjectConfig) {
		c.Filepath = path
	}
}

// Supplies a list of strings which act as the containers for KV pairs.
func WithPrefixes(prefixes ...string) Option {
	return func(c *LocalObjectConfig) {
		c.Prefixes = prefixes
	}
}
