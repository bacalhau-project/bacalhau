package types

type Provider interface {
	IsNotDisabled(kind string) bool
}
