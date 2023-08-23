package types

//go:generate go run gen_paths/generate.go
//go:generate go run gen_viper/generate.go
type BacalhauConfig struct {
	Node    NodeConfig
	User    UserConfig
	Metrics MetricsConfig
}

type UserConfig struct {
	KeyPath       string
	Libp2pKeyPath string
}

type MetricsConfig struct {
	Libp2pTracerPath string
	EventTracerPath  string
}
