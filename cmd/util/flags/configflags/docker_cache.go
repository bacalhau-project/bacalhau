package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/cfgtypes"
)

var DockerManifestCacheFlags = []Definition{
	{
		FlagName:             "docker-manifest-cache-size",
		ConfigPath:           cfgtypes.EnginesDockerManifestCacheSizeKey,
		DefaultValue:         cfgtypes.Default.Engines.Docker.ManifestCache.Size,
		Description:          `Specifies the number of items that can be held in the manifest cache`,
		EnvironmentVariables: []string{"BACALHAU_DOCKER_MANIFEST_CACHE_SIZE"},
	},
	{
		FlagName:             "docker-manifest-cache-duration",
		ConfigPath:           cfgtypes.EnginesDockerManifestCacheTTLKey,
		DefaultValue:         cfgtypes.Default.Engines.Docker.ManifestCache.TTL,
		Description:          `The default time-to-live for each record in the manifest cache`,
		EnvironmentVariables: []string{"BACALHAU_DOCKER_MANIFEST_CACHE_DURATION"},
	},
	{
		FlagName:             "docker-manifest-cache-frequency",
		ConfigPath:           cfgtypes.EnginesDockerManifestCacheRefreshKey,
		DefaultValue:         cfgtypes.Default.Engines.Docker.ManifestCache.Refresh,
		Description:          `The frequency that the checks for stale records is performed`,
		EnvironmentVariables: []string{"BACALHAU_DOCKER_MANIFEST_CACHE_FREQUENCY"},
	},
}
