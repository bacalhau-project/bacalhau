package cfgtypes

type Provider interface {
	Enabled(kind string) bool
}

type Configurable interface {
	Installed() bool
}
