package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

var DockerManifestCacheFlags = []Definition{
	{
		FlagName:             "docker-manifest-cache-size",
		ConfigPath:           types.EnginesDockerManifestCacheSizeKey,
		DefaultValue:         types.Default.Engines.Docker.ManifestCache.Size,
		Description:          `Specifies the number of items that can be held in the manifest cache`,
		EnvironmentVariables: []string{"BACALHAU_DOCKER_MANIFEST_CACHE_SIZE"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.EnginesDockerManifestCacheSizeKey),
	},
	{
		FlagName:             "docker-manifest-cache-duration",
		ConfigPath:           types.EnginesDockerManifestCacheTTLKey,
		DefaultValue:         types.Default.Engines.Docker.ManifestCache.TTL,
		Description:          `The default time-to-live for each record in the manifest cache`,
		EnvironmentVariables: []string{"BACALHAU_DOCKER_MANIFEST_CACHE_DURATION"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.EnginesDockerManifestCacheTTLKey),
	},
	{
		FlagName:             "docker-manifest-cache-frequency",
		ConfigPath:           types.EnginesDockerManifestCacheRefreshKey,
		DefaultValue:         types.Default.Engines.Docker.ManifestCache.Refresh,
		Description:          `The frequency that the checks for stale records is performed`,
		EnvironmentVariables: []string{"BACALHAU_DOCKER_MANIFEST_CACHE_FREQUENCY"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.EnginesDockerManifestCacheRefreshKey),
	},
}
