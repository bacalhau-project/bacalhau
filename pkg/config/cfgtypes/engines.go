package cfgtypes

import (
	"slices"
	"strings"
)

var _ Provider = (*EngineConfig)(nil)

type EngineConfig struct {
	Disabled []string `yaml:"Disabled,omitempty"`
	Docker   Docker   `yaml:"Docker,omitempty"`
	WASM     WASM     `yaml:"WASM,omitempty"`
}

func (e EngineConfig) Enabled(kind string) bool {
	return !slices.ContainsFunc(e.Disabled, func(s string) bool {
		return strings.ToLower(s) == strings.ToLower(kind)
	})
}

var _ Configurable = (*Docker)(nil)

// Docker represents the configuration settings for the Docker runtime provider.
type Docker struct {
	// ManifestCache specifies the settings for the Docker manifest cache.
	ManifestCache DockerManifestCache `yaml:"ManifestCache,omitempty"`
}

func (d Docker) Installed() bool {
	return d != Docker{}
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

var _ Configurable = (*WASM)(nil)

type WASM struct {
}

func (W WASM) Installed() bool {
	return true
}
