package model

import "fmt"

//go:generate stringer -type=Network --trimprefix=Network
type Network int

const (
	NetworkNone Network = iota
	NetworkFull
)

func ParseNetwork(s string) (Network, error) {
	for typ := NetworkNone; typ < NetworkFull; typ++ {
		if equal(typ.String(), s) {
			return typ, nil
		}
	}

	return NetworkNone, fmt.Errorf("%T: unknown type '%s'", NetworkNone, s)
}

func (n Network) MarshalText() ([]byte, error) {
	return []byte(n.String()), nil
}

func (n *Network) UnmarshalText(text []byte) (err error) {
	name := string(text)
	*n, err = ParseNetwork(name)
	return
}

type NetworkConfig struct {
	Type Network `json:"Type"`
}

// Returns whether network connections should be completely disabled according
// to this config.
func (n NetworkConfig) Disabled() bool {
	return n.Type == NetworkNone
}
