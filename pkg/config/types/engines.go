package types

import (
	"slices"
	"strings"
)

var _ Provider = (*EngineConfig)(nil)

type EngineConfig struct {
	// Disabled specifies a list of engines that are disabled.
	Disabled []string          `yaml:"Disabled,omitempty" json:"Disabled,omitempty"`
	Types    EngineConfigTypes `yaml:"Types,omitempty" json:"Types,omitempty"`
}

type EngineConfigTypes struct {
	Docker Docker `yaml:"Docker,omitempty" json:"Docker,omitempty"`
	WASM   WASM   `yaml:"WASM,omitempty" json:"WASM,omitempty"`
}

func (e EngineConfig) IsNotDisabled(kind string) bool {
	return !slices.ContainsFunc(e.Disabled, func(s string) bool {
		return strings.ToLower(s) == strings.ToLower(kind)
	})
}

// Docker represents the configuration settings for the Docker runtime provider.
type Docker struct {
	// ManifestCache specifies the settings for the Docker manifest cache.
	ManifestCache DockerManifestCache `yaml:"ManifestCache,omitempty" json:"ManifestCache,omitempty"`
}

// DockerManifestCache represents the configuration settings for the Docker manifest cache.
type DockerManifestCache struct {
	// Size specifies the size of the Docker manifest cache.
	Size uint64 `yaml:"Size,omitempty" json:"Size,omitempty"`
	// TTL specifies the time-to-live duration for cache entries.
	TTL Duration `yaml:"TTL,omitempty" json:"TTL,omitempty"`
	// Refresh specifies the refresh interval for cache entries.
	Refresh Duration `yaml:"Refresh,omitempty" json:"Refresh,omitempty"`
}

type WASM struct {
}
