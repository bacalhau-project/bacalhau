package v2

import (
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
