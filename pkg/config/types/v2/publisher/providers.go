package publisher

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/types/v2/types"
)

// Providers represents the configuration for different storage providers.
// It includes settings for IPFS, S3, Local, and HTTP providers.
type Providers struct {
	// IPFS is the configuration for the IPFS provider.
	IPFS IPFS
	// S3 is the configuration for the S3 provider.
	S3 S3
	// Local is the configuration for the Local provider.
	Local Local
	// HTTP is the configuration for the HTTP provider.
	HTTP HTTP

	LocalHTTPServer LocalHTTPServer
}

// IPFS represents the configuration settings for the IPFS storage provider.
type IPFS struct {
	// Enabled specifies whether the IPFS provider is enabled.
	Enabled bool
	// Endpoint specifies the endpoint Multiaddress for the IPFS provider.
	Endpoint string
}

// S3 represents the configuration settings for the S3 storage provider.
type S3 struct {
	// Enabled specifies whether the S3 provider is enabled.
	Enabled bool
	// Endpoint specifies the endpoint URL for the S3 provider.
	Endpoint string
	// AccessKey specifies the access key for the S3 provider.
	AccessKey string
	// SecretKey specifies the secret key for the S3 provider.
	SecretKey string
	// PreSignedURLEnabled specifies whether pre-signed URLs are enabled for the S3 provider.
	PreSignedURLEnabled bool
	// PreSignedURLExpiration specifies the duration before a pre-signed URL expires.
	PreSignedURLExpiration types.Duration
}

// Local represents the configuration settings for the Local storage provider.
type Local struct {
	// Enabled specifies whether the Local provider is enabled.
	Enabled bool
	// Volumes specifies the list of local volumes available for storage.
	Volumes []types.Volume
}

// HTTP represents the configuration settings for the HTTP storage provider.
type HTTP struct {
	// Enabled specifies whether the HTTP provider is enabled.
	Enabled bool
	// Endpoint specifies the endpoint URL for the HTTP provider.
	Endpoint string
	// Headers specifies the HTTP headers to be included in requests to the HTTP provider.
	Headers map[string]string
}

type LocalHTTPServer struct {
	Enabled bool
	Host    string
	Port    int
}
