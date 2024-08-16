package types

// Publisher represents the configuration for different publisher providers.
type Publisher struct {
	// IPFS is the configuration for the IPFS provider.
	IPFS IPFSPublisher `yaml:"IPFS,omitempty"`
	// S3 is the configuration for the S3 provider.
	S3 S3Publisher `yaml:"S3,omitempty"`
	// LocalHTTPServer is the configuration for the local http server run by the compute node
	LocalHTTPServer LocalHTTPServerPublisher
}

type IPFSPublisher struct {
	// Enabled specifies whether the IPFS provider is enabled.
	Enabled bool `yaml:"Enabled,omitempty"`
	// Endpoint specifies the endpoint Multiaddress for the IPFS provider.
	Endpoint string `yaml:"Endpoint,omitempty"`
}

type S3Publisher struct {
	// Enabled specifies whether the S3 provider is enabled.
	Enabled bool `yaml:"Enabled,omitempty"`
	// Endpoint specifies the endpoint URL for the S3 provider.
	Endpoint string `yaml:"Endpoint,omitempty"`
	// AccessKey specifies the access key for the S3 provider.
	AccessKey string `yaml:"AccessKey,omitempty"`
	// SecretKey specifies the secret key for the S3 provider.
	SecretKey string `yaml:"SecretKey,omitempty"`
	// PreSignedURLEnabled specifies whether pre-signed URLs are enabled for the S3 provider.
	PreSignedURLEnabled bool `yaml:"PreSignedURLEnabled,omitempty"`
	// PreSignedURLExpiration specifies the duration before a pre-signed URL expires.
	PreSignedURLExpiration Duration `yaml:"PreSignedURLExpiration,omitempty"`
}

type LocalHTTPServerPublisher struct {
	Enabled bool   `yaml:"Enabled,omitempty"`
	Host    string `yaml:"Host,omitempty"`
	Port    int    `yaml:"Port,omitempty"`
}
