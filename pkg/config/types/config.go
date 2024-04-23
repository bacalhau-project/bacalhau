package types

type NodeID string

//go:generate go run gen_paths/generate.go
//go:generate go run gen_viper/generate.go
type BacalhauConfig struct {
	ID      NodeID        `yaml:"ID"`
	Node    NodeConfig    `yaml:"Node"`
	User    UserConfig    `yaml:"User"`
	Metrics MetricsConfig `yaml:"Metrics"`
	Update  UpdateConfig  `yaml:"UpdateConfig"`
	Auth    AuthConfig    `yaml:"Auth"`
}

type UserConfig struct {
	KeyPath        string `yaml:"KeyPath"`
	Libp2pKeyPath  string `yaml:"Libp2PKeyPath"`
	InstallationID string `yaml:"InstallationID"`
}

type MetricsConfig struct {
	Libp2pTracerPath string `yaml:"Libp2PTracerPath"`
	EventTracerPath  string `yaml:"EventTracerPath"`
}

type UpdateConfig struct {
	SkipChecks     bool     `yaml:"SkipChecks"`
	CheckStatePath string   `yaml:"StatePath"`
	CheckFrequency Duration `yaml:"CheckFrequency"`
}

type ServerConfig struct {
	Address            string `yaml:"Address"`
	Port               uint16 `yaml:"Port"`
	AutoCertDomain     string `yaml:"AutoCertDomain"`
	AutoCertCache      string `yaml:"AutoCertCache"`
	TLSCertificateFile string `yaml:"TLSCertificateFile"`
	TLSKeyFile         string `yaml:"TLSKeyFile"`
	// These are TCP connection deadlines and not HTTP timeouts. They don't control the time it takes for our handlers
	// to complete. Deadlines operate on the connection, so our server will fail to return a result only after
	// the handlers try to access connection properties
	// ReadHeaderTimeout is the amount of time allowed to read request headers
	ReadHeaderTimeout Duration `yaml:"ReadHeaderTimeout"`
	// WriteTimeout is the maximum duration before timing out writes of the response.
	// It doesn't cancel the context and doesn't stop handlers from running even after failing the request.
	// It is for added safety and should be a bit longer than the request handler timeout for better error handling.
	WriteTimeout Duration `yaml:"WriteTimeout"`
}
