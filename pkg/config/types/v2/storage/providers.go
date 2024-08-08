package storage

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/types/v2/types"
)

// Providers represents the configuration for various storage providers.
// It includes settings for HTTP, IPFS, Local, and S3 providers.
type Providers struct {
	// HTTP is the configuration for the HTTP storage provider.
	HTTP HTTP
	// IPFS is the configuration for the IPFS storage provider.
	IPFS IPFS
	// Local is the configuration for the Local storage provider.
	Local Local
	// S3 is the configuration for the S3 storage provider.
	S3 S3
}

// IPFS represents the configuration settings for the IPFS storage provider.
type IPFS struct {
	// Enabled specifies whether the IPFS provider is enabled.
	Enabled bool
	// Endpoint specifies the endpoint URL for the IPFS provider.
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
