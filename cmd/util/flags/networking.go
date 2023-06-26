package flags

import (
	"github.com/spf13/pflag"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

func NewDefaultNetworkingFlagSettings() *NetworkingFlagSettings {
	return &NetworkingFlagSettings{
		Network: model.NetworkNone,
		Domains: []string{},
	}
}

type NetworkingFlagSettings struct {
	Network model.Network
	Domains []string
}

func NetworkingFlags(settings *NetworkingFlagSettings) *pflag.FlagSet {
	flags := pflag.NewFlagSet("Networking settings", pflag.ContinueOnError)
	flags.Var(
		NetworkFlag(&settings.Network),
		"network",
		`Networking capability required by the job. None, HTTP, or Full`,
	)
	flags.StringArrayVar(
		&settings.Domains,
		"domain",
		settings.Domains,
		`Domain(s) that the job needs to access (for HTTP networking)`,
	)
	return flags
}
