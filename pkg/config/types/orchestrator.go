package types

type Orchestrator struct {
	// Enabled indicates whether the orchestrator node is active and available for job submission.
	Enabled bool `yaml:"Enabled,omitempty"`
	// Host specifies the hostname or IP address on which the Orchestrator server listens for compute node connections.
	Host string `yaml:"Host,omitempty"`
	// Host specifies the port number on which the Orchestrator server listens for compute node connections.
	Port int `yaml:"Port,omitempty"`
	// Advertise specifies URL to advertise to other servers.
	Advertise string `yaml:"Advertise,omitempty"`
	// AuthSecret key specifies the key used by compute nodes to connect to an orchestrator.
	AuthSecret       string           `yaml:"AuthSecret,omitempty"`
	TLS              TLS              `yaml:"TLS,omitempty"`
	Cluster          Cluster          `yaml:"Cluster,omitempty"`
	NodeManager      NodeManager      `yaml:"NodeManager,omitempty"`
	Scheduler        Scheduler        `yaml:"Scheduler,omitempty"`
	EvaluationBroker EvaluationBroker `yaml:"EvaluationBroker,omitempty"`
}

type Cluster struct {
	// Name specifies the unique identifier for this orchestrator cluster.
	Name string `yaml:"Name,omitempty"`
	// Host specifies the hostname or IP address for cluster communication.
	Host string `yaml:"Host,omitempty"`
	// Port specifies the port number for cluster communication.
	Port int `yaml:"Port,omitempty"`
	// Advertise specifies the address to advertise to other cluster members.
	Advertise string `yaml:"Advertise,omitempty"`
	// Peers is a list of other cluster members to connect to on startup.
	Peers []string `yaml:"Peers,omitempty"`
}

type NodeManager struct {
	// DisconnectTimeout specifies how long to wait before considering a node disconnected.
	DisconnectTimeout Duration `yaml:"DisconnectTimeout,omitempty"`
	// ManualApproval, if true, requires manual approval for new compute nodes joining the cluster.
	ManualApproval bool `yaml:"ManualApproval,omitempty"`
}

type Scheduler struct {
	// WorkerCount specifies the number of concurrent workers for job scheduling.
	WorkerCount int `yaml:"WorkerCount,omitempty"`
	// HousekeepingInterval specifies how often to run housekeeping tasks.
	HousekeepingInterval Duration `yaml:"HousekeepingInterval,omitempty"`
	// HousekeepingTimeout specifies the maximum time allowed for a single housekeeping run.
	HousekeepingTimeout Duration `yaml:"HousekeepingTimeout,omitempty"`
}

type EvaluationBroker struct {
	// VisibilityTimeout specifies how long an evaluation can be claimed before it's returned to the queue.
	VisibilityTimeout Duration `yaml:"VisibilityTimeout,omitempty"`
	// MaxRetryCount specifies the maximum number of times an evaluation can be retried before being marked as failed.
	MaxRetryCount int `yaml:"MaxRetryCount,omitempty"`
}
