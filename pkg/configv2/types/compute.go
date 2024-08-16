package types

// Compute represents the configuration for the compute service on the Bacalhau node.
// It includes settings for enabling the service, connecting to orchestrators, TLS, heartbeat, store, capacity, and more.
type Compute struct {
	// Enabled when set to true will enable the compute service on the Bacalhau node.
	Enabled bool `yaml:"Enabled,omitempty"`
	// Orchestrators specifies a list of orchestrators the compute node will connect to.
	Orchestrators []string `yaml:"Orchestrators,omitempty"`
	// TLS specifies the TLS configuration used to connect to orchestrators.
	TLS TLS `yaml:"TLS,omitempty"`
	// Labels specifies a map of key value pairs the compute node will advertise to orchestrators.
	Labels map[string]string `yaml:"Labels,omitempty"`
	// Heartbeat specifies the compute node's heartbeat configuration.
	Heartbeat Heartbeat `yaml:"Heartbeat,omitempty"`
	// Capacity specifies the compute node's capacity configuration.
	Capacity Capacity `yaml:"Capacity,omitempty"`
	// Publishers specifies the configuration of publishers the compute node provides.
	Publishers Publisher `yaml:"Publishers,omitempty"`
	// Storages specifies the configuration of storages the compute node provides.
	Storages Storage `yaml:"Storages,omitempty"`
	// Engines specifies the configuration of engines the compute node provides.
	Engines Engine `yaml:"Engines,omitempty"`
	// Policy specifies the configuration of the compute node's job selection policy.
	Policy SelectionPolicy `yaml:"Policy,omitempty"`
}

// Heartbeat represents the configuration settings for the compute node's heartbeat messages.
type Heartbeat struct {
	// MessageInterval specifies the duration at which the compute node sends heartbeat messages to the orchestrators.
	MessageInterval Duration `yaml:"MessageInterval,omitempty"`
	// ResourceInterval specifies the duration at which the compute node sends resource messages to the orchestrators.
	ResourceInterval Duration `yaml:"ResourceInterval,omitempty"`
	// InfoInterval specifies the duration at which the compute node sends info messages to the orchestrators.
	InfoInterval Duration `yaml:"InfoInterval,omitempty"`
}

// Capacity represents the capacity configuration settings for the compute node.
type Capacity struct {
	// Total when specified overrides the auto-detected capacity of the compute node.
	// When provided, the Allocated capacity will be ignored.
	Total Resource `yaml:"Total,omitempty"`
	// Allocated specifies the percentage of the total capacity that can be allocated to jobs on the compute node.
	Allocated ResourceScaler `yaml:"Allocated,omitempty"`
}

// SelectionPolicy represents the job selection policy configuration for the compute node.
type SelectionPolicy struct {
	// Networked when set to true allows the compute node to accept jobs requiring network access.
	Networked bool `yaml:"Networked,omitempty"`
	// Local when set to true instructs the compute node to only accept jobs whose inputs it has locally.
	Local bool `yaml:"Local,omitempty"`
}
