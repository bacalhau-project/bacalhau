package types

// TLS represents the configuration settings for TLS (Transport Layer Security).
// It includes options for automatic certificate management as well as manually specified certificates.
type TLS struct {
	// AutoCert specifies a domain name for a certificate to be obtained via ACME (Automated Certificate Management Environment).
	AutoCert string `yaml:"AutoCert,omitempty"`
	// AutoCertCachePath specifies the path where the ACME client will cache certificates to avoid rate limits.
	AutoCertCachePath string `yaml:"AutoCertCachePath,omitempty"`
	// Certificate specifies the path to a TLS certificate file to be used.
	Certificate string `yaml:"Certificate,omitempty"`
	// Key specifies the path to the private key file corresponding to the TLS certificate.
	Key string `yaml:"Key,omitempty"`
}
