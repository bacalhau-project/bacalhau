package types

// Bacalhau represents the configuration for the Bacalhau system,
// including client, server, orchestrator, compute service, and telemetry settings.
type Bacalhau struct {
	// Repo specifies a path on the filesystem where Bacalhau will persist its state.
	Repo string `yaml:"Repo,omitempty"`
	// Name specifies the name the Bacalhau node will advertise on the network.
	Name string `yaml:"Name,omitempty"`
	// Client specifies the configuration of the Bacalhau client.
	Client Client `yaml:"Client,omitempty"`
	// Server specifies the configuration of the Bacalhau server.
	Server Server `yaml:"Server,omitempty"`
	// Orchestrator specifies the configuration of the Bacalhau orchestration service.
	Orchestrator Orchestrator `yaml:"Orchestrator,omitempty"`
	// Compute specifies the configuration of the Bacalhau compute service.
	Compute Compute `yaml:"Compute,omitempty"`
	// Telemetry specifies the configuration of the Bacalhau node's telemetry system.
	Telemetry Telemetry `yaml:"Telemetry,omitempty"`
}

func (b Bacalhau) Validate() error {
	//TODO implement me
	return nil
}

// Server represents the configuration settings for the Bacalhau server.
type Server struct {
	// Address specifies the endpoint Bacalhau will serve on.
	Address string `yaml:"Address,omitempty"`
	// Port specifies the port Bacalhau will serve on.
	Port int `yaml:"Port,omitempty"`
	// TLS specifies the TLS configuration for the server.
	TLS TLS `yaml:"TLS,omitempty"`
	// Auth specifies configuration for authorization and authentication.
	Auth AuthConfig `yaml:"Auth,omitempty"`
}

// Client represents the configuration settings for the Bacalhau client.
type Client struct {
	// Address specifies the endpoint the Bacalhau client will connect to a Bacalhau API on.
	Address string `yaml:"Address,omitempty"`
	// Certificate specifies the path of a certificate file (primarily for self-signed server certs) the client will use for API requests.
	Certificate string `yaml:"Certificate,omitempty"`
	// Insecure when true instructs the client not to verify the certificate of the server.
	Insecure bool `yaml:"Insecure,omitempty"`
}

// WebUI represents the configuration settings for the Bacalhau WebUI.
type WebUI struct {
	// Enabled when true enables the WebUI on the Bacalhau node.
	Enabled bool `yaml:"Enabled,omitempty"`
	// Server specifies the configuration of the server for the WebUI.
	Server Server `yaml:"Server,omitempty"`
}

// JobDownloaders represents the configuration settings for job downloaders in Bacalhau.
type JobDownloaders struct {
	// Timeout specifies the duration after which job downloads will time out.
	Timeout Duration `yaml:"Timeout,omitempty"`
	// IPFS specifies the configuration for the IPFS job downloader.
	IPFS IPFSStorage `yaml:"IPFS,omitempty"`
	// S3 specifies the configuration for the S3 job downloader.
	S3 S3Storage `yaml:"S3,omitempty"`
}
