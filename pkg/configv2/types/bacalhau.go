package types

type Bacalhau struct {
	API                 API                 `yaml:"API,omitempty"`
	NameProvider        string              `yaml:"NameProvider,omitempty"`
	DataDir             string              `yaml:"DataDir,omitempty"`
	StrictVersionMatch  bool                `yaml:"StrictVersionMatch,omitempty"`
	Orchestrator        Orchestrator        `yaml:"Orchestrator,omitempty"`
	Compute             Compute             `yaml:"Compute,omitempty"`
	WebUI               WebUI               `yaml:"WebUI,omitempty"`
	InputSources        InputSourcesConfig  `yaml:"InputSources,omitempty"`
	Publishers          PublishersConfig    `yaml:"Publishers,omitempty"`
	Executors           ExecutorsConfig     `yaml:"Executors,omitempty"`
	ResultDownloaders   ResultDownloaders   `yaml:"ResultDownloaders,omitempty"`
	JobDefaults         JobDefaults         `yaml:"JobDefaults,omitempty"`
	JobAdmissionControl JobAdmissionControl `yaml:"JobAdmissionControl,omitempty"`
	Logging             Logging             `yaml:"Logging,omitempty"`
	UpdateConfig        UpdateConfig        `yaml:"UpdateConfig,omitempty"`
	FeatureFlags        FeatureFlags        `yaml:"FeatureFlags,omitempty"`
}

type UpdateConfig struct {
	Interval Duration `yaml:"Interval,omitempty"`
}

type FeatureFlags struct {
	ExecTranslation bool `yaml:"ExecTranslation,omitempty"`
}

type API struct {
	Address string     `yaml:"Address,omitempty"`
	TLS     TLS        `yaml:"TLS,omitempty"`
	Auth    AuthConfig `yaml:"Auth,omitempty"`
}

type TLS struct {
	CertFile string `yaml:"Certificate,omitempty"`
	KeyFile  string `yaml:"Key,omitempty"`
	CAFile   string `yaml:"CAFile,omitempty"`
}

type WebUI struct {
	Enabled bool   `yaml:"Enabled,omitempty"`
	Listen  string `yaml:"Listen,omitempty"`
}

type Logging struct {
	Level                string   `yaml:"Level,omitempty"`
	Mode                 string   `yaml:"Mode,omitempty"`
	LogDebugInfoInterval Duration `yaml:"LogDebugInfoInterval,omitempty"`
}

type JobAdmissionControl struct {
	RejectStatelessJobs bool   `yaml:"RejectStatelessJobs,omitempty"`
	AcceptNetworkedJobs bool   `yaml:"AcceptNetworkedJobs,omitempty"`
	ProbeHTTP           string `yaml:"ProbeHTTP,omitempty"`
	ProbeExec           string `yaml:"ProbeExec,omitempty"`
}

type ResultDownloaders struct {
	Timeout Duration                     `yaml:"Timeout,omitempty"`
	Config  map[string]map[string]string `yaml:"Config,omitempty"`
}

type JobDefaults struct {
	Batch   JobDefaultsConfig `yaml:"Batch,omitempty"`
	Daemon  JobDefaultsConfig `yaml:"Daemon,omitempty"`
	Service JobDefaultsConfig `yaml:"Service,omitempty"`
	Ops     JobDefaultsConfig `yaml:"Ops,omitempty"`
}

type JobDefaultsConfig struct {
	Priority int               `yaml:"Priority,omitempty"`
	Task     TaskDefaultConfig `yaml:"Task,omitempty"`
}

type TaskDefaultConfig struct {
	Resources Resource               `yaml:"Resources,omitempty"`
	Publisher DefaultPublisherConfig `yaml:"Publisher,omitempty"`
	Timeouts  TaskTimeoutConfig      `yaml:"Timeouts,omitempty"`
}

type DefaultPublisherConfig struct {
	Type string `yaml:"Type,omitempty"`
}

type TaskTimeoutConfig struct {
	ExecutionTimeout Duration `yaml:"ExecutionTimeout,omitempty"`
}
