package local

type Option func(*LocalObjectConfig)

func WithDataFolder(path string) Option {
	return func(c *LocalObjectConfig) {
		c.Path = path
	}
}

// Supplies a list of strings which act as the containers for KV pairs.
func WithPrefixes(prefixes ...string) Option {
	return func(c *LocalObjectConfig) {
		c.Prefixes = prefixes
	}
}
