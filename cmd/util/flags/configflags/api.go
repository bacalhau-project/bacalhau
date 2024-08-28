package configflags

import (
	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
)

var ClientAPIFlags = []Definition{
	{
		FlagName:     "api-host",
		DefaultValue: types2.Default.API.Host,
		ConfigPath:   "API.Host",
		Description: `The host for the client and server to communicate on (via REST).
Ignored if BACALHAU_API_HOST environment variable is set.`,
		EnvironmentVariables: []string{"BACALHAU_API_HOST"},
	},
	{
		FlagName:     "api-port",
		DefaultValue: types2.Default.API.Port,
		ConfigPath:   "API.Port",
		Description: `The port for the client and server to communicate on (via REST).
Ignored if BACALHAU_API_PORT environment variable is set.`,
		EnvironmentVariables: []string{"BACALHAU_API_PORT"},
	},
	{
		FlagName:             "tls",
		DefaultValue:         types2.Default.API.TLS.UseTLS,
		ConfigPath:           "API.TLS.UseTLS",
		Description:          `Instructs the client to use TLS`,
		EnvironmentVariables: []string{"BACALHAU_API_TLS"},
	},
	{
		FlagName:     "cacert",
		DefaultValue: types2.Default.API.TLS.CAFile,
		ConfigPath:   "API.TLS.CAFile",
		Description: `The location of a CA certificate file when self-signed certificates
	are used by the server`,
		EnvironmentVariables: []string{"BACALHAU_API_CACERT"},
	},
	{
		FlagName:             "insecure",
		DefaultValue:         types2.Default.API.TLS.Insecure,
		ConfigPath:           "API.TLS.Insecure",
		Description:          `Enables TLS but does not verify certificates`,
		EnvironmentVariables: []string{"BACALHAU_API_INSECURE"},
	},
}

var ServerAPIFlags = []Definition{
	{
		FlagName:             "port",
		DefaultValue:         types2.Default.API.Port,
		ConfigPath:           "API.Port",
		Description:          `The port to server on.`,
		EnvironmentVariables: []string{"BACALHAU_SERVER_PORT"},
	},
	{
		FlagName:             "host",
		DefaultValue:         types2.Default.API.Host,
		ConfigPath:           "API.Host",
		Description:          `The host to serve on.`,
		EnvironmentVariables: []string{"BACALHAU_SERVER_HOST"},
	},
}

var RequesterTLSFlags = []Definition{
	{
		FlagName:     "autocert",
		DefaultValue: types2.Default.API.TLS.AutoCert,
		ConfigPath:   "API.TLS.AutoCert",
		Description: `Specifies a host name for which ACME is used to obtain a TLS Certificate.
Using this option results in the API serving over HTTPS`,
		EnvironmentVariables: []string{"BACALHAU_AUTO_TLS"},
	},
	{
		FlagName:             "tlscert",
		DefaultValue:         types2.Default.API.TLS.CertFile,
		ConfigPath:           "API.TLS.CertFile",
		Description:          `Specifies a TLS certificate file to be used by the requester node`,
		EnvironmentVariables: []string{"BACALHAU_TLS_CERT"},
	},
	{
		FlagName:             "tlskey",
		DefaultValue:         types2.Default.API.TLS.KeyFile,
		ConfigPath:           "API.TLS.KeyFile",
		Description:          `Specifies a TLS key file matching the certificate to be used by the requester node`,
		EnvironmentVariables: []string{"BACALHAU_TLS_KEY"},
	},
	{
		FlagName:             "self-signed",
		DefaultValue:         types2.Default.API.TLS.SelfSigned,
		ConfigPath:           "API.TLS.SelfSigned",
		Description:          `Specifies whether to auto-generate a self-signed certificate for the requester node`,
		EnvironmentVariables: []string{"BACALHAU_TLS_SELFSIGNED"},
	},
}
