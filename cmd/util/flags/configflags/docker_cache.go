package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

var DockerManifestCacheFlags = []Definition{
	{
		FlagName:             "docker-manifest-cache-size",
		ConfigPath:           types.EnginesTypesDockerManifestCacheSizeKey,
		DefaultValue:         config.Default.Engines.Types.Docker.ManifestCache.Size,
		Description:          `Specifies the number of items that can be held in the manifest cache`,
		EnvironmentVariables: []string{"BACALHAU_DOCKER_MANIFEST_CACHE_SIZE"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.EnginesTypesDockerManifestCacheSizeKey),
	},
	{
		FlagName:             "docker-manifest-cache-duration",
		ConfigPath:           types.EnginesTypesDockerManifestCacheTTLKey,
		DefaultValue:         config.Default.Engines.Types.Docker.ManifestCache.TTL,
		Description:          `The default time-to-live for each record in the manifest cache`,
		EnvironmentVariables: []string{"BACALHAU_DOCKER_MANIFEST_CACHE_DURATION"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.EnginesTypesDockerManifestCacheTTLKey),
	},
	{
		FlagName:             "docker-manifest-cache-frequency",
		ConfigPath:           types.EnginesTypesDockerManifestCacheRefreshKey,
		DefaultValue:         config.Default.Engines.Types.Docker.ManifestCache.Refresh,
		Description:          `The frequency that the checks for stale records is performed`,
		EnvironmentVariables: []string{"BACALHAU_DOCKER_MANIFEST_CACHE_FREQUENCY"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.EnginesTypesDockerManifestCacheRefreshKey),
	},
}
