package types

import (
	"slices"
	"strings"
)

type ExecutorsConfig struct {
	Disabled []string                          `yaml:"Disabled,omitempty"`
	Config   map[string]map[string]interface{} `yaml:"Config,omitempty"`
}

func (e ExecutorsConfig) Enabled(kind string) bool {
	return !slices.ContainsFunc(e.Disabled, func(s string) bool {
		return strings.ToLower(s) == strings.ToLower(kind)
	})
}

func (e ExecutorsConfig) HasConfig(kind string) bool {
	_, ok := e.Config[kind]
	return ok
}

func (e ExecutorsConfig) ConfigMap() map[string]map[string]interface{} {
	return e.Config
}

// Docker represents the configuration settings for the Docker runtime provider.
type Docker struct {
	// ManifestCache specifies the settings for the Docker manifest cache.
	ManifestCache DockerManifestCache
}

// DockerManifestCache represents the configuration settings for the Docker manifest cache.
type DockerManifestCache struct {
	// Size specifies the size of the Docker manifest cache.
	Size uint64
	// TTL specifies the time-to-live duration for cache entries.
	TTL Duration
	// Refresh specifies the refresh interval for cache entries.
	Refresh Duration
}

const KindExecutorDocker = "Docker"

func (d Docker) Kind() string {
	return KindExecutorDocker
}

// wasm doesn't have a config, but may still be enabled or disabled, so we need this constant.
const KindExecutorWASM = "WASM"
