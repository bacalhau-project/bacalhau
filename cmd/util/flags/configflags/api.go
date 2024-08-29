package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/cfgtypes"
)

var ClientAPIFlags = []Definition{
	{
		FlagName:     "api-host",
		DefaultValue: cfgtypes.Default.API.Host,
		ConfigPath:   cfgtypes.APIHostKey,
		Description: `The host for the client and server to communicate on (via REST).
Ignored if BACALHAU_API_HOST environment variable is set.`,
		EnvironmentVariables: []string{"BACALHAU_API_HOST"},
	},
	{
		FlagName:     "api-port",
		DefaultValue: cfgtypes.Default.API.Port,
		ConfigPath:   cfgtypes.APIPortKey,
		Description: `The port for the client and server to communicate on (via REST).
Ignored if BACALHAU_API_PORT environment variable is set.`,
		EnvironmentVariables: []string{"BACALHAU_API_PORT"},
	},
	{
		FlagName:             "tls",
		DefaultValue:         cfgtypes.Default.API.TLS.UseTLS,
		ConfigPath:           cfgtypes.APITLSUseTLSKey,
		Description:          `Instructs the client to use TLS`,
		EnvironmentVariables: []string{"BACALHAU_API_TLS"},
	},
	{
		FlagName:     "cacert",
		DefaultValue: cfgtypes.Default.API.TLS.CAFile,
		ConfigPath:   cfgtypes.APITLSCAFileKey,
		Description: `The location of a CA certificate file when self-signed certificates
	are used by the server`,
		EnvironmentVariables: []string{"BACALHAU_API_CACERT"},
	},
	{
		FlagName:             "insecure",
		DefaultValue:         cfgtypes.Default.API.TLS.Insecure,
		ConfigPath:           cfgtypes.APITLSInsecureKey,
		Description:          `Enables TLS but does not verify certificates`,
		EnvironmentVariables: []string{"BACALHAU_API_INSECURE"},
	},
}

var ServerAPIFlags = []Definition{
	{
		FlagName:             "port",
		DefaultValue:         cfgtypes.Default.API.Port,
		ConfigPath:           cfgtypes.APIPortKey,
		Description:          `The port to server on.`,
		EnvironmentVariables: []string{"BACALHAU_SERVER_PORT"},
	},
	{
		FlagName:             "host",
		DefaultValue:         cfgtypes.Default.API.Host,
		ConfigPath:           cfgtypes.APIHostKey,
		Description:          `The host to serve on.`,
		EnvironmentVariables: []string{"BACALHAU_SERVER_HOST"},
	},
}

var RequesterTLSFlags = []Definition{
	{
		FlagName:     "autocert",
		DefaultValue: cfgtypes.Default.API.TLS.AutoCert,
		ConfigPath:   cfgtypes.APITLSAutoCertKey,
		Description: `Specifies a host name for which ACME is used to obtain a TLS Certificate.
Using this option results in the API serving over HTTPS`,
		EnvironmentVariables: []string{"BACALHAU_AUTO_TLS"},
	},
	{
		FlagName:             "tlscert",
		DefaultValue:         cfgtypes.Default.API.TLS.CertFile,
		ConfigPath:           cfgtypes.APITLSCertFileKey,
		Description:          `Specifies a TLS certificate file to be used by the requester node`,
		EnvironmentVariables: []string{"BACALHAU_TLS_CERT"},
	},
	{
		FlagName:             "tlskey",
		DefaultValue:         cfgtypes.Default.API.TLS.KeyFile,
		ConfigPath:           cfgtypes.APITLSKeyFileKey,
		Description:          `Specifies a TLS key file matching the certificate to be used by the requester node`,
		EnvironmentVariables: []string{"BACALHAU_TLS_KEY"},
	},
	{
		FlagName:             "self-signed",
		DefaultValue:         cfgtypes.Default.API.TLS.SelfSigned,
		ConfigPath:           cfgtypes.APITLSSelfSignedKey,
		Description:          `Specifies whether to auto-generate a self-signed certificate for the requester node`,
		EnvironmentVariables: []string{"BACALHAU_TLS_SELFSIGNED"},
	},
}
