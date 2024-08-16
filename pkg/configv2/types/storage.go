package types

// Storage represents the configuration for various storage providers.
// It includes settings for HTTP, IPFS, Local, and S3 providers.
type Storage struct {
	// HTTP is the configuration for the HTTP storage provider.
	HTTP HTTPStorage
	// IPFS is the configuration for the IPFS storage provider.
	IPFS IPFSStorage `yaml:"IPFS,omitempty"`
	// Local is the configuration for the Local storage provider.
	Local LocalStorage `yaml:"Local,omitempty"`
	// S3 is the configuration for the S3 storage provider.
	S3 S3Storage `yaml:"S3,omitempty"`
}

type HTTPStorage struct {
	// Enabled specifies whether the HTTP provider is enabled.
	Enabled bool `yaml:"Enabled,omitempty"`
}

type IPFSStorage struct {
	// Enabled specifies whether the IPFS provider is enabled.
	Enabled bool `yaml:"Enabled,omitempty"`
	// Endpoint specifies the endpoint URL for the IPFS provider.
	Endpoint string `yaml:"Endpoint,omitempty"`
}

type S3Storage struct {
	// Enabled specifies whether the S3 provider is enabled.
	Enabled bool `yaml:"Enabled,omitempty"`
	// Endpoint specifies the endpoint URL for the S3 provider.
	Endpoint string `yaml:"Endpoint,omitempty"`
	// AccessKey specifies the access key for the S3 provider.
	AccessKey string `yaml:"AccessKey,omitempty"`
	// SecretKey specifies the secret key for the S3 provider.
	SecretKey string `yaml:"SecretKey,omitempty"`
}

type LocalStorage struct {
	// Enabled specifies whether the Local provider is enabled.
	Enabled bool `yaml:"Enabled,omitempty"`
	// Volumes specifies the list of local volumes available for storage.
	Volumes []Volume `yaml:"Volumes,omitempty"`
}
