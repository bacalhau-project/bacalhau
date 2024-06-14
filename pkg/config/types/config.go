package types

//go:generate go run gen_paths/generate.go
//go:generate go run gen_viper/generate.go
type BacalhauConfig struct {
	Node    NodeConfig    `yaml:"Node"`
	User    UserConfig    `yaml:"User"`
	Metrics MetricsConfig `yaml:"Metrics"`
	Update  UpdateConfig  `yaml:"UpdateConfig"`
	Auth    AuthConfig    `yaml:"Auth"`
}

type UserConfig struct {
	KeyPath        string `yaml:"KeyPath"`
	InstallationID string `yaml:"InstallationID"`
}

type MetricsConfig struct {
	EventTracerPath string `yaml:"EventTracerPath"`
}

type UpdateConfig struct {
	SkipChecks     bool     `yaml:"SkipChecks"`
	CheckStatePath string   `yaml:"StatePath"`
	CheckFrequency Duration `yaml:"CheckFrequency"`
}
