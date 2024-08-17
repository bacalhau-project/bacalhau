package types

type Compute struct {
	Enabled       bool     `yaml:"Enabled,omitempty"`
	Orchestrators []string `yaml:"Orchestrators,omitempty"`
	TLS           TLS      `yaml:"TLS,omitempty"`

	// TODO what is this?
	Authorization interface{} `yaml:"Authorization,omitempty"`

	Heartbeat         Heartbeat         `yaml:"Heartbeat,omitempty"`
	Labels            map[string]string `yaml:"Labels,omitempty"`
	TotalCapacity     Resource          `yaml:"TotalCapacity,omitempty"`
	AllocatedCapacity ResourceScaler    `yaml:"AllocatedCapacity,omitempty"`
	StateStore        ComputeStateStore `yaml:"StateStore,omitempty"`
	Volumes           []Volume          `yaml:"Volumes,omitempty"`
}

type Heartbeat struct {
	InfoUpdateInterval     Duration `yaml:"InfoUpdateInterval,omitempty"`
	ResourceUpdateInterval Duration `yaml:"ResourceUpdateInterval,omitempty"`
	Interval               Duration `yaml:"Interval,omitempty"`
}

type Capacity struct {
	Total     Resource       `yaml:"Total,omitempty"`
	Allocated ResourceScaler `yaml:"Allocated,omitempty"`
}

type ComputeStateStore struct {
	ExecutionGCInterval  Duration     `yaml:"ExecutionGCInterval,omitempty"`
	ExecutionGCThreshold Duration     `yaml:"ExecutionGCThreshold,omitempty"`
	Backend              StoreBackend `yaml:"Backend,omitempty"`
}

type Volume struct {
	Name      string `yaml:"Name,omitempty"`
	Path      string `yaml:"Path,omitempty"`
	ReadWrite bool   `yaml:"Write,omitempty"`
}
