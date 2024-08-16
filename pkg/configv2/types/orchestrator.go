package types

// Orchestrator represents the configuration for the orchestration service on the Bacalhau node.
// It includes settings for enabling the service, network endpoints, TLS configuration, and various subsystems.
type Orchestrator struct {
	// Enabled specifies whether the orchestration service is enabled on the Bacalhau node.
	Enabled bool `yaml:"Enabled,omitempty"`
	// Listen specifies the address the orchestration service will listen on for connections from compute nodes.
	Listen string `yaml:"Listen,omitempty"`
	// Port specifies the port the orchestration service will listen for connections from compute nodes.
	Port int `yaml:"Port,omitempty"`
	// Advertise specifies the endpoint the orchestration service will advertise to the network for connections from compute nodes.
	Advertise string `yaml:"Advertise,omitempty"`
	// AuthSecret is a secret string that clients must use to connect. NATS servers
	// must supply this value, while clients can also supply it as the user part
	// of their Orchestrator URL.
	AuthSecret string `yaml:"AuthSecret,omitempty"`
	// TLS specifies the TLS configuration of the orchestration service.
	TLS TLS `yaml:"TLS,omitempty"`
	// Cluster specifies the cluster configuration of the orchestration service.
	Cluster Cluster `yaml:"Cluster,omitempty"`
	// NodeManager specifies the node manager configuration of the orchestration service.
	NodeManager NodeManager `yaml:"NodeManager,omitempty"`
	// Scheduler specifies the scheduler configuration of the orchestration service.
	Scheduler Scheduler `yaml:"Scheduler,omitempty"`
	// Broker specifies the evaluation broker configuration of the orchestration service.
	Broker EvaluationBroker `yaml:"Broker,omitempty"`
}

// Cluster represents the configuration settings for the orchestration service NATs cluster.
type Cluster struct {
	// Name specifies the name of the cluster the orchestration service will connect to.
	Name string `yaml:"Name,omitempty"`
	// Listen specifies the address the orchestration service will listen on for connections from other orchestration services.
	Listen string `yaml:"Listen,omitempty"`
	// Port specifies the port the orchestration service will listen on for connections from other orchestration services.
	Port int `yaml:"Port,omitempty"`
	// Advertise specifies the endpoint the orchestration service will advertise to the network for connections from other orchestration services.
	Advertise string `yaml:"Advertise,omitempty"`
	// Peers specifies the list of peer orchestration services.
	Peers []string `yaml:"Peers,omitempty"`
	// TLS specifies the TLS configuration for connections from orchestration services.
	TLS TLS `yaml:"TLS,omitempty"`
}

// NodeManager represents the configuration settings for the node manager within the orchestration service.
type NodeManager struct {
	// DisconnectTimeout specifies the duration after which nodes will be considered disconnected if no heartbeat message is received.
	DisconnectTimeout Duration `yaml:"DisconnectTimeout,omitempty"`
	// AutoApprove specifies whether to automatically approve a node's membership when a connection is established.
	// When set to false, sets a node's approval status to 'pending' when a connection is established.
	AutoApprove bool `yaml:"AutoApprove,omitempty"`
}

// Scheduler represents the configuration settings for the scheduler within the orchestration service.
type Scheduler struct {
	// Workers specifies the number of workers the scheduler will run.
	Workers int `yaml:"Workers,omitempty"`
	// HousekeepingInterval specifies the interval at which housekeeping tasks are performed.
	HousekeepingInterval Duration `yaml:"HousekeepingInterval,omitempty"`
	// HousekeepingTimeout specifies the timeout duration for housekeeping tasks.
	HousekeepingTimeout Duration `yaml:"HousekeepingTimeout,omitempty"`
}

// EvaluationBroker represents the configuration settings for the evaluation broker within the orchestration service.
type EvaluationBroker struct {
	// VisibilityTimeout specifies the duration after which an unprocessed evaluation is re-queued.
	VisibilityTimeout Duration `yaml:"VisibilityTimeout,omitempty"`
	// MaxRetries specifies the maximum number of retries for processing an evaluation.
	MaxRetries int `yaml:"MaxRetries,omitempty"`
}
