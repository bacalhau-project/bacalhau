package types

type ConfigProvider interface {
	Enabled(kind string) bool
	HasConfig(kind string) bool
	ConfigMap() map[string]map[string]interface{}
}

type ProviderType interface {
	Kind() string
}
