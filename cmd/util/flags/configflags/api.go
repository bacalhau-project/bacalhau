package configflags

import "github.com/bacalhau-project/bacalhau/pkg/config/types"

var ClientAPIFlags = []Definition{
	{
		FlagName:     "api-host",
		DefaultValue: Default.Node.ClientAPI.Host,
		ConfigPath:   types.NodeClientAPIHost,
		Description: `The host for the client and server to communicate on (via REST).
Ignored if BACALHAU_API_HOST environment variable is set.`,
		EnvironmentVariables: []string{"BACALHAU_API_HOST"},
	},
	{
		FlagName:     "api-port",
		DefaultValue: Default.Node.ClientAPI.Port,
		ConfigPath:   types.NodeClientAPIPort,
		Description: `The port for the client and server to communicate on (via REST).
Ignored if BACALHAU_API_PORT environment variable is set.`,
		EnvironmentVariables: []string{"BACALHAU_API_PORT"},
	},
	{
		FlagName:             "tls",
		DefaultValue:         Default.Node.ClientAPI.ClientTLS.UseTLS,
		ConfigPath:           types.NodeClientAPIClientTLSUseTLS,
		Description:          `Instructs the client to use TLS`,
		EnvironmentVariables: []string{"BACALHAU_API_TLS"},
	},
	{
		FlagName:     "cacert",
		DefaultValue: Default.Node.ClientAPI.ClientTLS.CACert,
		ConfigPath:   types.NodeClientAPIClientTLSCACert,
		Description: `The location of a CA certificate file when self-signed certificates
	are used by the server`,
		EnvironmentVariables: []string{"BACALHAU_API_CACERT"},
	},
	{
		FlagName:             "insecure",
		DefaultValue:         Default.Node.ClientAPI.ClientTLS.Insecure,
		ConfigPath:           types.NodeClientAPIClientTLSInsecure,
		Description:          `Enables TLS but does not verify certificates`,
		EnvironmentVariables: []string{"BACALHAU_API_INSECURE"},
	},
}

var ServerAPIFlags = []Definition{
	{
		FlagName:             "port",
		DefaultValue:         Default.Node.ServerAPI.Port,
		ConfigPath:           types.NodeServerAPIPort,
		Description:          `The port to server on.`,
		EnvironmentVariables: []string{"BACALHAU_SERVER_PORT"},
	},
	{
		FlagName:             "host",
		DefaultValue:         Default.Node.ServerAPI.Host,
		ConfigPath:           types.NodeServerAPIHost,
		Description:          `The host to serve on.`,
		EnvironmentVariables: []string{"BACALHAU_SERVER_HOST"},
	},
}

var RequesterTLSFlags = []Definition{
	{
		FlagName:     "autocert",
		DefaultValue: Default.Node.ServerAPI.TLS.AutoCert,
		ConfigPath:   types.NodeServerAPITLSAutoCert,
		Description: `Specifies a host name for which ACME is used to obtain a TLS Certificate.
Using this option results in the API serving over HTTPS`,
		EnvironmentVariables: []string{"BACALHAU_AUTO_TLS"},
	},
	{
		FlagName:             "tlscert",
		DefaultValue:         Default.Node.ServerAPI.TLS.ServerCertificate,
		ConfigPath:           types.NodeServerAPITLSServerCertificate,
		Description:          `Specifies a TLS certificate file to be used by the requester node`,
		EnvironmentVariables: []string{"BACALHAU_TLS_CERT"},
	},
	{
		FlagName:             "tlskey",
		DefaultValue:         Default.Node.ServerAPI.TLS.ServerKey,
		ConfigPath:           types.NodeServerAPITLSServerKey,
		Description:          `Specifies a TLS key file matching the certificate to be used by the requester node`,
		EnvironmentVariables: []string{"BACALHAU_TLS_KEY"},
	},
}
