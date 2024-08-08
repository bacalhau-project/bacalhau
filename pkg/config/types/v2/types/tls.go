package types

// TLS represents the configuration settings for TLS (Transport Layer Security).
// It includes options for automatic certificate management as well as manually specified certificates.
type TLS struct {
	// AutoCert specifies a domain name for a certificate to be obtained via ACME (Automated Certificate Management Environment).
	AutoCert string
	// AutoCertCachePath specifies the path where the ACME client will cache certificates to avoid rate limits.
	AutoCertCachePath string

	// Certificate specifies the path to a TLS certificate file to be used.
	Certificate string
	// Key specifies the path to the private key file corresponding to the TLS certificate.
	Key string
}
