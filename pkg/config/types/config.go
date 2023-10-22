package types

//go:generate go run gen_paths/generate.go
//go:generate go run gen_viper/generate.go
type BacalhauConfig struct {
	Node    NodeConfig    `yaml:"Node"`
	User    UserConfig    `yaml:"User"`
	Metrics MetricsConfig `yaml:"Metrics"`
	Update  UpdateConfig  `yaml:"UpdateConfig"`
}

type UserConfig struct {
	KeyPath       string `yaml:"KeyPath"`
	Libp2pKeyPath string `yaml:"Libp2PKeyPath"`
	UserID        string `yaml:"UserID"`
}

type MetricsConfig struct {
	Libp2pTracerPath string `yaml:"Libp2PTracerPath"`
	EventTracerPath  string `yaml:"EventTracerPath"`
}

type UpdateConfig struct {
	SkipChecks     bool     `yaml:"SkipChecks"`
	CheckStatePath string   `yaml:"StatePath"`
	CheckFrequency Duration `yaml:"CheckFrequency"`
}
