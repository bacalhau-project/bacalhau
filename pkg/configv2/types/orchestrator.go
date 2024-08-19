package types

type Orchestrator struct {
	Enabled          bool             `yaml:"Enabled,omitempty"`
	Listen           string           `yaml:"Listen,omitempty"`
	Advertise        string           `yaml:"Advertise,omitempty"`
	TLS              TLS              `yaml:"TLS,omitempty"`
	Cluster          Cluster          `yaml:"Cluster,omitempty"`
	NodeManager      NodeManager      `yaml:"NodeManager,omitempty"`
	Scheduler        Scheduler        `yaml:"Scheduler,omitempty"`
	EvaluationBroker EvaluationBroker `yaml:"EvaluationBroker,omitempty"`
}

type Cluster struct {
	Listen    string   `yaml:"Listen,omitempty"`
	Advertise string   `yaml:"Advertise,omitempty"`
	TLS       TLS      `yaml:"TLS,omitempty"`
	Peers     []string `yaml:"Peers,omitempty"`
}

type NodeManager struct {
	GCThreshold       Duration `yaml:"GCThreshold,omitempty"`
	GCInterval        Duration `yaml:"GCInterval,omitempty"`
	DisconnectTimeout Duration `yaml:"DisconnectTimeout,omitempty"`
	ManualApproval    bool     `yaml:"ManualApproval,omitempty"`
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
