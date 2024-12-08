package types

type Orchestrator struct {
	// Enabled indicates whether the orchestrator node is active and available for job submission.
	Enabled bool `yaml:"Enabled,omitempty" json:"Enabled,omitempty"`
	// Host specifies the hostname or IP address on which the Orchestrator server listens for compute node connections.
	Host string `yaml:"Host,omitempty" json:"Host,omitempty"`
	// Host specifies the port number on which the Orchestrator server listens for compute node connections.
	Port int `yaml:"Port,omitempty" json:"Port,omitempty"`
	// Advertise specifies URL to advertise to other servers.
	Advertise string `yaml:"Advertise,omitempty" json:"Advertise,omitempty"`
	// Auth specifies the authentication configuration for compute nodes to connect to the orchestrator.
	Auth OrchestratorAuth `yaml:"Auth,omitempty" json:"Auth,omitempty"`
	// TLS specifies the TLS related configuration on the orchestrator for when compute nodes need to connect.
	TLS              OrchestratorTLS  `yaml:"TLS,omitempty" json:"TLS,omitempty"`
	Cluster          Cluster          `yaml:"Cluster,omitempty" json:"Cluster,omitempty"`
	NodeManager      NodeManager      `yaml:"NodeManager,omitempty" json:"NodeManager,omitempty"`
	Scheduler        Scheduler        `yaml:"Scheduler,omitempty" json:"Scheduler,omitempty"`
	EvaluationBroker EvaluationBroker `yaml:"EvaluationBroker,omitempty" json:"EvaluationBroker,omitempty"`
	// SupportReverseProxy configures the orchestrator node to run behind a reverse proxy
	SupportReverseProxy bool `yaml:"SupportReverseProxy,omitempty" json:"SupportReverseProxy,omitempty"`
}

type OrchestratorAuth struct {
	// Token specifies the key for compute nodes to be able to access the orchestrator
	Token string `yaml:"Token,omitempty" json:"Token,omitempty"`
}

type OrchestratorTLS struct {
	// ServerKey specifies the private key file path given to NATS server to serve TLS connections.
	ServerKey string `yaml:"ServerKey,omitempty" json:"ServerKey,omitempty"`
	// ServerCert specifies the certificate file path given to NATS server to serve TLS connections.
	ServerCert string `yaml:"ServerCert,omitempty" json:"ServerCert,omitempty"`
	// ServerTimeout specifies the TLS timeout, in seconds, set on the NATS server.
	ServerTimeout int `yaml:"ServerTimeout,omitempty" json:"ServerTimeout,omitempty"`

	// CACert specifies the CA file path that the orchestrator node trusts when connecting to NATS server.
	CACert string `yaml:"CACert,omitempty" json:"CACert,omitempty"`
}

type Cluster struct {
	// Name specifies the unique identifier for this orchestrator cluster.
	Name string `yaml:"Name,omitempty" json:"Name,omitempty"`
	// Host specifies the hostname or IP address for cluster communication.
	Host string `yaml:"Host,omitempty" json:"Host,omitempty"`
	// Port specifies the port number for cluster communication.
	Port int `yaml:"Port,omitempty" json:"Port,omitempty"`
	// Advertise specifies the address to advertise to other cluster members.
	Advertise string `yaml:"Advertise,omitempty" json:"Advertise,omitempty"`
	// Peers is a list of other cluster members to connect to on startup.
	Peers []string `yaml:"Peers,omitempty" json:"Peers,omitempty"`
}

type NodeManager struct {

	// DisconnectTimeout specifies how long to wait before considering a node disconnected.
	DisconnectTimeout Duration `yaml:"DisconnectTimeout,omitempty" json:"DisconnectTimeout,omitempty"`
	// ManualApproval, if true, requires manual approval for new compute nodes joining the cluster.
	ManualApproval bool `yaml:"ManualApproval,omitempty" json:"ManualApproval,omitempty"`
}

type Scheduler struct {
	// WorkerCount specifies the number of concurrent workers for job scheduling.
	WorkerCount int `yaml:"WorkerCount,omitempty" json:"WorkerCount,omitempty"`
	// QueueBackoff specifies the time to wait before retrying a failed job.
	QueueBackoff Duration `yaml:"QueueBackoff,omitempty" json:"QueueBackoff,omitempty"`
	// HousekeepingInterval specifies how often to run housekeeping tasks.
	HousekeepingInterval Duration `yaml:"HousekeepingInterval,omitempty" json:"HousekeepingInterval,omitempty"`
	// HousekeepingTimeout specifies the maximum time allowed for a single housekeeping run.
	HousekeepingTimeout Duration `yaml:"HousekeepingTimeout,omitempty" json:"HousekeepingTimeout,omitempty"`
}

type EvaluationBroker struct {
	// VisibilityTimeout specifies how long an evaluation can be claimed before it's returned to the queue.
	VisibilityTimeout Duration `yaml:"VisibilityTimeout,omitempty" json:"VisibilityTimeout,omitempty"`
	// MaxRetryCount specifies the maximum number of times an evaluation can be retried before being marked as failed.
	MaxRetryCount int `yaml:"MaxRetryCount,omitempty" json:"MaxRetryCount,omitempty"`
}
