package types

type Engine struct {
	// Docker is the configuration for the Docker runtime provider.
	Docker Docker `yaml:"Docker,omitempty"`
	// WASM is the configuration for the WASM runtime provider.
	WASM WASM `yaml:"WASM,omitempty"`
}

// Docker represents the configuration settings for the Docker runtime provider.
type Docker struct {
	// Enabled specifies whether the Docker provider is enabled.
	Enabled bool `yaml:"Enabled,omitempty"`
	// TODO the docker engine doesn't permit this field to be configured yet.
	// Endpoint specifies the endpoint URI for the Docker daemon.
	// Endpoint string `yaml:"Endpoint,omitempty"`
	// ManifestCache specifies the settings for the Docker manifest cache.
	ManifestCache DockerManifestCache `yaml:"ManifestCache,omitempty"`
}

// DockerManifestCache represents the configuration settings for the Docker manifest cache.
type DockerManifestCache struct {
	// Size specifies the size of the Docker manifest cache.
	Size uint64 `yaml:"Size,omitempty"`
	// TTL specifies the time-to-live duration for cache entries.
	TTL Duration `yaml:"TTL,omitempty"`
	// Refresh specifies the refresh interval for cache entries.
	Refresh Duration `yaml:"Refresh,omitempty"`
}

// WASM represents the configuration settings for the WASM runtime provider.
type WASM struct {
	// Enabled specifies whether the WASM provider is enabled.
	Enabled bool `yaml:"Enabled,omitempty"`
}
