package types

type Orchestrator struct {
	Enabled   bool   `yaml:"Enabled,omitempty"`
	Listen    string `yaml:"Listen,omitempty"`
	Advertise string `yaml:"Advertise,omitempty"`
	TLS       TLS    `yaml:"TLS,omitempty"`
	// TODO what is this for?
	Authorization interface{} `yaml:"Authorization,omitempty"`

	Cluster          Cluster                `yaml:"Cluster,omitempty"`
	NodeManager      NodeManager            `yaml:"NodeManager,omitempty"`
	StateStore       OrchestratorStateStore `yaml:"StateStore,omitempty"`
	Scheduler        Scheduler              `yaml:"Scheduler,omitempty"`
	EvaluationBroker EvaluationBroker       `yaml:"EvaluationBroker,omitempty"`
}

type Cluster struct {
	Listen    string `yaml:"Listen,omitempty"`
	Advertise string `yaml:"Advertise,omitempty"`
	TLS       TLS    `yaml:"TLS,omitempty"`

	// TODO don't know what this is for yet.
	Authorization interface{} `yaml:"Authorization,omitempty"`

	Peers []string `yaml:"Peers,omitempty"`
}

type NodeManager struct {
	GCThreshold       Duration `yaml:"GCThreshold,omitempty"`
	GCInterval        Duration `yaml:"GCInterval,omitempty"`
	DisconnectTimeout Duration `yaml:"DisconnectTimeout,omitempty"`
	ManualApproval    bool     `yaml:"ManualApproval,omitempty"`
}

type OrchestratorStateStore struct {
	JobGCInterval   Duration     `yaml:"JobGCInterval,omitempty"`
	JobGCThreshold  Duration     `yaml:"JobGCThreshold,omitempty"`
	EvalGCThreshold Duration     `yaml:"EvalGCThreshold,omitempty"`
	Backend         StoreBackend `yaml:"Backend,omitempty"`
}

type Scheduler struct {
	WorkerCount          int      `yaml:"WorkerCount,omitempty"`
	HousekeepingInterval Duration `yaml:"HousekeepingInterval,omitempty"`
	HousekeepingTimeout  Duration `yaml:"HousekeepingTimeout,omitempty"`
}

type EvaluationBroker struct {
	VisibilityTimeout Duration `yaml:"VisibilityTimeout,omitempty"`
	MaxRetryCount     int      `yaml:"MaxRetryCount,omitempty"`
}
