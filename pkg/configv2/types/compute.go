package types

type Compute struct {
	Enabled               bool              `yaml:"Enabled,omitempty"`
	Orchestrators         []string          `yaml:"Orchestrators,omitempty"`
	TLS                   TLS               `yaml:"TLS,omitempty"`
	Heartbeat             Heartbeat         `yaml:"Heartbeat,omitempty"`
	Labels                map[string]string `yaml:"Labels,omitempty"`
	AllocatedCapacity     ResourceScaler    `yaml:"AllocatedCapacity,omitempty"`
	AllowListedLocalPaths []string          `yaml:"AllowListedLocalPaths"`
}

type Heartbeat struct {
	InfoUpdateInterval     Duration `yaml:"InfoUpdateInterval,omitempty"`
	ResourceUpdateInterval Duration `yaml:"ResourceUpdateInterval,omitempty"`
	Interval               Duration `yaml:"Interval,omitempty"`
}

type Volume struct {
	Name      string `yaml:"Name,omitempty"`
	Path      string `yaml:"Path,omitempty"`
	ReadWrite bool   `yaml:"Write,omitempty"`
}
