package objectstore

type Option func(*DBConfig)

func WithLocation(loc string) Option {
	return func(c *DBConfig) {
		c.Location = loc
	}
}
