package executor

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/types/v2/types"
)

// Providers represents the configuration for different runtime providers.
// It includes settings for Docker and WASM providers.
type Providers struct {
	// Docker is the configuration for the Docker runtime provider.
	Docker Docker
	// WASM is the configuration for the WASM runtime provider.
	WASM WASM
}

// Docker represents the configuration settings for the Docker runtime provider.
type Docker struct {
	// Enabled specifies whether the Docker provider is enabled.
	Enabled bool
	// Endpoint specifies the endpoint URI for the Docker daemon.
	Endpoint string
	// ManifestCache specifies the settings for the Docker manifest cache.
	ManifestCache DockerManifestCache
}

// DockerManifestCache represents the configuration settings for the Docker manifest cache.
type DockerManifestCache struct {
	// Size specifies the size of the Docker manifest cache.
	Size uint64
	// TTL specifies the time-to-live duration for cache entries.
	TTL types.Duration
	// Refresh specifies the refresh interval for cache entries.
	Refresh types.Duration
}

// WASM represents the configuration settings for the WASM runtime provider.
type WASM struct {
	// Enabled specifies whether the WASM provider is enabled.
	Enabled bool
}
