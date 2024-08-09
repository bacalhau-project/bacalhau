package v2

import (
	"github.com/bacalhau-project/bacalhau/pkg/authn"
	"github.com/bacalhau-project/bacalhau/pkg/config/types/v2/storage"
	"github.com/bacalhau-project/bacalhau/pkg/config/types/v2/types"
)

// Bacalhau represents the configuration for the Bacalhau system,
// including client, server, orchestrator, compute service, and telemetry settings.
type Bacalhau struct {
	// Repo specifies a path on the filesystem where Bacalhau will persist its state.
	Repo string
	// Name specifies the name the Bacalhau node will advertise on the network.
	Name string
	// Client specifies the configuration of the Bacalhau client.
	Client Client
	// Server specifies the configuration of the Bacalhau server.
	Server Server
	// Orchestrator specifies the configuration of the Bacalhau orchestration service.
	Orchestrator Orchestrator
	// Compute specifies the configuration of the Bacalhau compute service.
	Compute Compute
	// Telemetry specifies the configuration of the Bacalhau node's telemetry system.
	Telemetry Telemetry
}

// Server represents the configuration settings for the Bacalhau server.
type Server struct {
	// Address specifies the endpoint Bacalhau will serve on.
	Address string
	// Port specifies the port Bacalhau will serve on.
	Port int
	// TLS specifies the TLS configuration for the server.
	TLS types.TLS

	Auth AuthConfig
}

// Client represents the configuration settings for the Bacalhau client.
type Client struct {
	// Address specifies the endpoint the Bacalhau client will connect to a Bacalhau API on.
	Address string
	// Certificate specifies the path of a certificate file (primarily for self-signed server certs) the client will use for API requests.
	Certificate string
	// Insecure when true instructs the client not to verify the certificate of the server.
	Insecure bool
}

// WebUI represents the configuration settings for the Bacalhau WebUI.
type WebUI struct {
	// Enabled when true enables the WebUI on the Bacalhau node.
	Enabled bool
	// Server specifies the configuration of the server for the WebUI.
	Server Server
}

// JobDownloaders represents the configuration settings for job downloaders in Bacalhau.
type JobDownloaders struct {
	// Timeout specifies the duration after which job downloads will time out.
	Timeout types.Duration
	// IPFS specifies the configuration for the IPFS job downloader.
	IPFS storage.IPFS
	// S3 specifies the configuration for the S3 job downloader.
	S3 storage.S3
	// HTTP specifies the configuration for the HTTP job downloader.
	HTTP storage.HTTP
}

// AuthenticatorConfig is config for a specific named authentication method,
// specifying the type of authentication and the path to a policy file that
// controls the method. Some implementation types may require policies that meet
// a certain interface beyond the default – see the documentation on that type
// for more info.
type AuthenticatorConfig struct {
	Type       authn.MethodType `yaml:"Type"`
	PolicyPath string           `yaml:"PolicyPath,omitempty"`
}

// AuthConfig is config that controls user authentication and authorization.
type AuthConfig struct {
	// TokensPath is the location where a state file of tokens will be stored.
	// By default it will be local to the Bacalhau repo, but can be any location
	// in the host filesystem. Tokens are sensitive and should be stored in a
	// location that is only readable to the current user.
	TokensPath string `yaml:"TokensPath"`

	// Methods maps "method names" to authenticator implementations. A method
	// name is a human-readable string chosen by the person configuring the
	// system that is shown to users to help them pick the authentication method
	// they want to use. There can be multiple usages of the same Authenticator
	// *type* but with different configs and parameters, each identified with a
	// unique method name.
	//
	// For example, if an implementation wants to allow users to log in with
	// Github or Bitbucket, they might both use an authenticator implementation
	// of type "oidc", and each would appear once on this provider with key /
	// method name "github" and "bitbucket".
	//
	// By default, only a single authentication method that accepts
	// authentication via client keys will be enabled.
	Methods map[string]AuthenticatorConfig `yaml:"Methods"`

	// AccessPolicyPath is the path to a file or directory that will be loaded as
	// the policy to apply to all inbound API requests. If unspecified, a policy
	// that permits access to all API endpoints to both authenticated and
	// unauthenticated users (the default as of v1.2.0) will be used.
	AccessPolicyPath string `yaml:"AccessPolicyPath"`
}
