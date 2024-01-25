package configflags

import "github.com/bacalhau-project/bacalhau/pkg/config/types"

var DockerManifestCacheFlags = []Definition{
	{
		FlagName:             "docker-manifest-cache-size",
		ConfigPath:           types.NodeComputeManifestCacheSize,
		DefaultValue:         Default.Node.Compute.ManifestCache.Size,
		Description:          `Specifies the number of items that can be held in the manifest cache`,
		EnvironmentVariables: []string{"BACALHAU_DOCKER_MANIFEST_CACHE_SIZE"},
	},
	{
		FlagName:             "docker-manifest-cache-duration",
		ConfigPath:           types.NodeComputeManifestCacheDuration,
		DefaultValue:         Default.Node.Compute.ManifestCache.Duration,
		Description:          `The default time-to-live for each record in the manifest cache`,
		EnvironmentVariables: []string{"BACALHAU_DOCKER_MANIFEST_CACHE_DURATION"},
	},
	{
		FlagName:             "docker-manifest-cache-frequency",
		ConfigPath:           types.NodeComputeManifestCacheFrequency,
		DefaultValue:         Default.Node.Compute.ManifestCache.Frequency,
		Description:          `The frequency that the checks for stale records is performed`,
		EnvironmentVariables: []string{"BACALHAU_DOCKER_MANIFEST_CACHE_FREQUENCY"},
	},
}
