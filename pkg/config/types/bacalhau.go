package types

// NB: Developers, after making changes (comments included) to this struct or any of its children, run go generate.

//go:generate go run gen/generate.go ./
//go:generate go fmt ./generated_constants.go ./generated_descriptions.go
type Bacalhau struct {
	API API `yaml:"API,omitempty"`
	// NameProvider specifies the method used to generate names for the node. One of: hostname, aws, gcp, uuid, puuid.
	NameProvider string `yaml:"NameProvider,omitempty"`
	// DataDir specifies a location on disk where the bacalhau node will maintain state.
	DataDir string `yaml:"DataDir,omitempty"`
	// StrictVersionMatch indicates whether to enforce strict version matching.
	StrictVersionMatch  bool                `yaml:"StrictVersionMatch,omitempty"`
	Orchestrator        Orchestrator        `yaml:"Orchestrator,omitempty"`
	Compute             Compute             `yaml:"Compute,omitempty"`
	WebUI               WebUI               `yaml:"WebUI,omitempty"`
	InputSources        InputSourcesConfig  `yaml:"InputSources,omitempty"`
	Publishers          PublishersConfig    `yaml:"Publishers,omitempty"`
	Engines             EngineConfig        `yaml:"Engines,omitempty"`
	ResultDownloaders   ResultDownloaders   `yaml:"ResultDownloaders,omitempty"`
	JobDefaults         JobDefaults         `yaml:"JobDefaults,omitempty"`
	JobAdmissionControl JobAdmissionControl `yaml:"JobAdmissionControl,omitempty"`
	Logging             Logging             `yaml:"Logging,omitempty"`
	UpdateConfig        UpdateConfig        `yaml:"UpdateConfig,omitempty"`
	FeatureFlags        FeatureFlags        `yaml:"FeatureFlags,omitempty"`
	// DisableAnalytics when set to true disables bacalhau from sharing anonymous user data with Expanso.
	DisableAnalytics bool `yaml:"DisableAnalytics,omitempty"`
}

type API struct {
	// Host specifies the hostname or IP address on which the API server listens or the client connects.
	Host string `yaml:"Host,omitempty"`
	// Port specifies the port number on which the API server listens or the client connects.
	Port int        `yaml:"Port,omitempty"`
	TLS  TLS        `yaml:"TLS,omitempty"`
	Auth AuthConfig `yaml:"Auth,omitempty"`
}

type TLS struct {
	// CertFile specifies the path to the TLS certificate file.
	CertFile string `yaml:"CertFile,omitempty"`
	// KeyFile specifies the path to the TLS private key file.
	KeyFile string `yaml:"KeyFile,omitempty"`
	// CAFile specifies the path to the Certificate Authority file.
	CAFile string `yaml:"CAFile,omitempty"`

	// UseTLS indicates whether to use TLS for client connections.
	UseTLS bool `yaml:"UseTLS,omitempty"`
	// Insecure allows insecure TLS connections (e.g., self-signed certificates).
	Insecure bool `yaml:"Insecure"`

	// SelfSigned indicates whether to use a self-signed certificate.
	SelfSigned bool `yaml:"SelfSigned,omitempty"`
	// AutoCert specifies the domain for automatic certificate generation.
	AutoCert string `yaml:"AutoCert,omitempty"`
	// AutoCertCachePath specifies the directory to cache auto-generated certificates.
	AutoCertCachePath string `yaml:"AutoCertCachePath,omitempty"`
}

type WebUI struct {
	// Enabled indicates whether the Web UI is enabled.
	Enabled bool `yaml:"Enabled,omitempty"`
	// Listen specifies the address and port on which the Web UI listens.
	Listen string `yaml:"Listen,omitempty"`
}

type Logging struct {
	// Level sets the logging level. One of: trace, debug, info, warn, error, fatal, panic.
	Level string `yaml:"Level,omitempty"`
	// Mode specifies the logging mode. One of: default, json.
	Mode string `yaml:"Mode,omitempty"`
	// LogDebugInfoInterval specifies the interval for logging debug information.
	LogDebugInfoInterval Duration `yaml:"LogDebugInfoInterval,omitempty"`
}

type FeatureFlags struct {
	// ExecTranslation enables the execution translation feature.
	ExecTranslation bool `yaml:"ExecTranslation,omitempty"`
}

type UpdateConfig struct {
	// Interval specifies the time between update checks, when set to 0 update checks are not performed.
	Interval Duration `yaml:"Interval,omitempty"`
}

type JobAdmissionControl struct {
	// RejectStatelessJobs indicates whether to reject stateless jobs, i.e. jobs without inputs.
	RejectStatelessJobs bool `yaml:"RejectStatelessJobs,omitempty"`
	// AcceptNetworkedJobs indicates whether to accept jobs that require network access.
	AcceptNetworkedJobs bool `yaml:"AcceptNetworkedJobs,omitempty"`
	// ProbeHTTP specifies the HTTP endpoint for probing job submission.
	ProbeHTTP string `yaml:"ProbeHTTP,omitempty"`
	// ProbeExec specifies the command to execute for probing job submission.
	ProbeExec string `yaml:"ProbeExec,omitempty"`
}
