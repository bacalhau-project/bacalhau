package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

var ClientAPIFlags = []Definition{
	{
		FlagName:     "api-host",
		DefaultValue: config.Default.API.Host,
		ConfigPath:   types.APIHostKey,
		Description: `The host for the client and server to communicate on (via REST).
Ignored if BACALHAU_API_HOST environment variable is set.`,
		EnvironmentVariables: []string{"BACALHAU_API_HOST"},
	},
	{
		FlagName:     "api-port",
		DefaultValue: config.Default.API.Port,
		ConfigPath:   types.APIPortKey,
		Description: `The port for the client and server to communicate on (via REST).
Ignored if BACALHAU_API_PORT environment variable is set.`,
		EnvironmentVariables: []string{"BACALHAU_API_PORT"},
	},
	{
		FlagName:             "tls",
		DefaultValue:         config.Default.API.TLS.UseTLS,
		ConfigPath:           types.APITLSUseTLSKey,
		Description:          `Instructs the client to use TLS`,
		EnvironmentVariables: []string{"BACALHAU_API_TLS"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.APITLSUseTLSKey),
	},
	{
		FlagName:     "cacert",
		DefaultValue: config.Default.API.TLS.CAFile,
		ConfigPath:   types.APITLSCAFileKey,
		Description: `The location of a CA certificate file when self-signed certificates
	are used by the server`,
		EnvironmentVariables: []string{"BACALHAU_API_CACERT"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.APITLSCAFileKey),
	},
	{
		FlagName:             "insecure",
		DefaultValue:         config.Default.API.TLS.Insecure,
		ConfigPath:           types.APITLSInsecureKey,
		Description:          `Enables TLS but does not verify certificates`,
		EnvironmentVariables: []string{"BACALHAU_API_INSECURE"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.APITLSInsecureKey),
	},
}

var ServerAPIFlags = []Definition{
	{
		FlagName:             "port",
		DefaultValue:         config.Default.API.Port,
		ConfigPath:           types.APIPortKey,
		Description:          `The port to server on.`,
		EnvironmentVariables: []string{"BACALHAU_SERVER_PORT"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.APIPortKey),
	},
	{
		FlagName:             "host",
		DefaultValue:         config.Default.API.Host,
		ConfigPath:           types.APIHostKey,
		Description:          `The host to serve on.`,
		EnvironmentVariables: []string{"BACALHAU_SERVER_HOST"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.APIHostKey),
	},
}

var RequesterTLSFlags = []Definition{
	{
		FlagName:     "autocert",
		DefaultValue: config.Default.API.TLS.AutoCert,
		ConfigPath:   types.APITLSAutoCertKey,
		Description: `Specifies a host name for which ACME is used to obtain a TLS Certificate.
Using this option results in the API serving over HTTPS`,
		EnvironmentVariables: []string{"BACALHAU_AUTO_TLS"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.APITLSAutoCertKey),
	},
	{
		FlagName:             "tlscert",
		DefaultValue:         config.Default.API.TLS.CertFile,
		ConfigPath:           types.APITLSCertFileKey,
		Description:          `Specifies a TLS certificate file to be used by the requester node`,
		EnvironmentVariables: []string{"BACALHAU_TLS_CERT"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.APITLSCertFileKey),
	},
	{
		FlagName:             "tlskey",
		DefaultValue:         config.Default.API.TLS.KeyFile,
		ConfigPath:           types.APITLSKeyFileKey,
		Description:          `Specifies a TLS key file matching the certificate to be used by the requester node`,
		EnvironmentVariables: []string{"BACALHAU_TLS_KEY"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.APITLSKeyFileKey),
	},
	{
		FlagName:             "self-signed",
		DefaultValue:         config.Default.API.TLS.SelfSigned,
		ConfigPath:           types.APITLSSelfSignedKey,
		Description:          `Specifies whether to auto-generate a self-signed certificate for the requester node`,
		EnvironmentVariables: []string{"BACALHAU_TLS_SELFSIGNED"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.APITLSSelfSignedKey),
	},
}
