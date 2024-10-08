package types

type API struct {
	// Host specifies the hostname or IP address on which the API server listens or the client connects.
	Host string `yaml:"Host,omitempty" json:"Host,omitempty"`
	// Port specifies the port number on which the API server listens or the client connects.
	Port int        `yaml:"Port,omitempty" json:"Port,omitempty"`
	TLS  TLS        `yaml:"TLS,omitempty" json:"TLS,omitempty"`
	Auth AuthConfig `yaml:"Auth,omitempty" json:"Auth,omitempty"`
}

type TLS struct {
	// CertFile specifies the path to the TLS certificate file.
	CertFile string `yaml:"CertFile,omitempty" json:"CertFile,omitempty"`
	// KeyFile specifies the path to the TLS private key file.
	KeyFile string `yaml:"KeyFile,omitempty" json:"KeyFile,omitempty"`
	// CAFile specifies the path to the Certificate Authority file.
	CAFile string `yaml:"CAFile,omitempty" json:"CAFile,omitempty"`

	// UseTLS indicates whether to use TLS for client connections.
	UseTLS bool `yaml:"UseTLS,omitempty" json:"UseTLS,omitempty"`
	// Insecure allows insecure TLS connections (e.g., self-signed certificates).
	Insecure bool `yaml:"Insecure" json:"Insecure"`

	// SelfSigned indicates whether to use a self-signed certificate.
	SelfSigned bool `yaml:"SelfSigned,omitempty" json:"SelfSigned,omitempty"`
	// AutoCert specifies the domain for automatic certificate generation.
	AutoCert string `yaml:"AutoCert,omitempty" json:"AutoCert,omitempty"`
	// AutoCertCachePath specifies the directory to cache auto-generated certificates.
	AutoCertCachePath string `yaml:"AutoCertCachePath,omitempty" json:"AutoCertCachePath,omitempty"`
}
