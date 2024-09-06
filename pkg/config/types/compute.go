package types

type Compute struct {
	// Enabled indicates whether the compute node is active and available for job execution.
	Enabled bool `yaml:"Enabled,omitempty"`
	// Orchestrators specifies a list of orchestrator endpoints that this compute node connects to.
	Orchestrators []string  `yaml:"Orchestrators,omitempty"`
	TLS           TLS       `yaml:"TLS,omitempty"`
	Heartbeat     Heartbeat `yaml:"Heartbeat,omitempty"`
	// Labels are key-value pairs used to describe and categorize the compute node.
	Labels            map[string]string `yaml:"Labels,omitempty"`
	AllocatedCapacity ResourceScaler    `yaml:"AllocatedCapacity,omitempty"`
	// AllowListedLocalPaths specifies a list of local file system paths that the compute node is allowed to access.
	AllowListedLocalPaths []string `yaml:"AllowListedLocalPaths"`
}

type Heartbeat struct {
	// InfoUpdateInterval specifies the time between updates of non-resource information to the orchestrator.
	InfoUpdateInterval Duration `yaml:"InfoUpdateInterval,omitempty"`
	// ResourceUpdateInterval specifies the time between updates of resource information to the orchestrator.
	ResourceUpdateInterval Duration `yaml:"ResourceUpdateInterval,omitempty"`
	// Interval specifies the time between heartbeat signals sent to the orchestrator.
	Interval Duration `yaml:"Interval,omitempty"`
}
