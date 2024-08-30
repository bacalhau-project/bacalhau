package types

type Orchestrator struct {
	Enabled          bool             `yaml:"Enabled,omitempty"`
	Host             string           `yaml:"Host,omitempty"`
	Port             int              `yaml:"Port,omitempty"`
	Advertise        string           `yaml:"Advertise,omitempty"`
	AuthSecret       string           `yaml:"AuthSecret,omitempty"`
	TLS              TLS              `yaml:"TLS,omitempty"`
	Cluster          Cluster          `yaml:"Cluster,omitempty"`
	NodeManager      NodeManager      `yaml:"NodeManager,omitempty"`
	Scheduler        Scheduler        `yaml:"Scheduler,omitempty"`
	EvaluationBroker EvaluationBroker `yaml:"EvaluationBroker,omitempty"`
}

type Cluster struct {
	Name      string   `yaml:"Name,omitempty"`
	Host      string   `yaml:"Host,omitempty"`
	Port      int      `yaml:"Port,omitempty"`
	Advertise string   `yaml:"Advertise,omitempty"`
	Peers     []string `yaml:"Peers,omitempty"`
}

type NodeManager struct {
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
