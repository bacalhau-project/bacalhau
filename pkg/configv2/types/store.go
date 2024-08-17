package types

type StoreBackend struct {
	Type   string            `yaml:"Type,omitempty"`
	Config map[string]string `yaml:"Config,omitempty"`
}
