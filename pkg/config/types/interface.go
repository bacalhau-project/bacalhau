package types

type Provider interface {
	IsNotDisabled(kind string) bool
}

type Configurable interface {
	IsConfigured() bool
}
