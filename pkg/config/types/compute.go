package types

type Compute struct {
	// Enabled indicates whether the compute node is active and available for job execution.
	Enabled bool `yaml:"Enabled,omitempty" json:"Enabled,omitempty"`
	// Orchestrators specifies a list of orchestrator endpoints that this compute node connects to.
	Orchestrators []string `yaml:"Orchestrators,omitempty" json:"Orchestrators,omitempty"`
	// Auth specifies the authentication configuration for compute nodes to connect to the orchestrator.
	Auth              ComputeAuth    `yaml:"Auth,omitempty" json:"Auth,omitempty"`
	Heartbeat         Heartbeat      `yaml:"Heartbeat,omitempty" json:"Heartbeat,omitempty"`
	AllocatedCapacity ResourceScaler `yaml:"AllocatedCapacity,omitempty" json:"AllocatedCapacity,omitempty"`
	// AllowListedLocalPaths specifies a list of local file system paths that the compute node is allowed to access.
	AllowListedLocalPaths []string `yaml:"AllowListedLocalPaths" json:"AllowListedLocalPaths,omitempty"`
	// TLS specifies the TLS related configuration on the compute node when connecting with the orchestrator.
	TLS ComputeTLS `yaml:"TLS,omitempty" json:"TLS,omitempty"`
	// Env specifies environment variable configuration for the compute node
	Env EnvConfig `yaml:"Env,omitempty" json:"Env,omitempty"`
}

type ComputeAuth struct {
	// Token specifies the key for compute nodes to be able to access the orchestrator.
	Token string `yaml:"Token,omitempty" json:"Token,omitempty"`
}

type ComputeTLS struct {
	// CACert specifies the CA file path that the compute node trusts when connecting to orchestrator.
	CACert string `yaml:"CACert,omitempty" json:"CACert,omitempty"`

	// RequireTLS specifies if the compute node enforces encrypted communication with orchestrator.
	RequireTLS bool `yaml:"RequireTLS,omitempty" json:"RequireTLS,omitempty"`
}

type Heartbeat struct {
	// InfoUpdateInterval specifies the time between updates of non-resource information to the orchestrator.
	InfoUpdateInterval Duration `yaml:"InfoUpdateInterval,omitempty" json:"InfoUpdateInterval,omitempty"`
	// Deprecated: use Interval instead
	ResourceUpdateInterval Duration `yaml:"ResourceUpdateInterval,omitempty" json:"ResourceUpdateInterval,omitempty"`
	// Interval specifies the time between heartbeat signals sent to the orchestrator.
	Interval Duration `yaml:"Interval,omitempty" json:"Interval,omitempty"`
}

// EnvConfig specifies environment variable configuration for the compute node
type EnvConfig struct {
	// AllowList specifies which host environment variables can be forwarded to jobs.
	// Supports glob patterns (e.g., "AWS_*", "API_*")
	AllowList []string `yaml:"AllowList,omitempty" json:"AllowList,omitempty"`
}
