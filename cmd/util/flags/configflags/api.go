package configflags

import (
	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
)

var APIFlags = []Definition{
	// NB(forrest): replaces api-host, host, api-port, and port since we are unable to bind those flags to fields of the config.
	{
		FlagName:             "api-address",
		ConfigPath:           "API.Address",
		DefaultValue:         types2.Default.API.Address,
		Description:          `The address for the client and server to communicate on (via REST).`,
		EnvironmentVariables: []string{"BACALHAU_API"},
	},
}
