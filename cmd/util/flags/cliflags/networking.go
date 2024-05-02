package cliflags

import (
	"github.com/spf13/pflag"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func NewDefaultNetworkingFlagSettings() *NetworkingFlagSettings {
	return &NetworkingFlagSettings{
		Network: models.NetworkNone,
		Domains: []string{},
	}
}

type NetworkingFlagSettings struct {
	Network models.Network
	Domains []string
}

func NetworkingFlags(settings *NetworkingFlagSettings) *pflag.FlagSet {
	flagset := pflag.NewFlagSet("Networking settings", pflag.ContinueOnError)
	flagset.Var(
		flags.NetworkFlag(&settings.Network),
		"network",
		`Networking capability required by the job. None, HTTP, or Full`,
	)
	flagset.StringArrayVar(
		&settings.Domains,
		"domain",
		settings.Domains,
		`Domain(s) that the job needs to access (for HTTP networking)`,
	)
	return flagset
}
