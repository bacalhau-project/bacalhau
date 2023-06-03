package docker

import (
	"strconv"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/cache"
	"github.com/bacalhau-project/bacalhau/pkg/cache/basic"
	"github.com/bacalhau-project/bacalhau/pkg/util"
)

//nolint:unused
var DockerTagCache cache.Cache[string]

//nolint:unused
var DockerManifestCache cache.Cache[ImageManifest]

const DefaultCacheSize = uint64(1000)
const DefaultCacheDuration = time.Hour

const tagCacheSizeEnvVar = "DOCKER_TAG_CACHE_SIZE"
const tagCacheDurationEnvVar = "DOCKER_TAG_CACHE_DURATION"

const manifestCacheSizeEnvVar = "DOCKER_MANIFEST_CACHE_SIZE"
const manifestCacheDurationEnvVar = "DOCKER_MANIFEST_CACHE_DURATION"

func init() { //nolint:gochecknoinits
	tagCacheDuration := util.GetEnvAs[time.Duration](
		tagCacheDurationEnvVar, DefaultCacheDuration, time.ParseDuration,
	)
	manifestCacheDuration := util.GetEnvAs[time.Duration](
		manifestCacheDurationEnvVar, DefaultCacheDuration, time.ParseDuration,
	)

	tagCacheSize := util.GetEnvAs[uint64](
		tagCacheSizeEnvVar, DefaultCacheSize, func(k string) (uint64, error) {
			return strconv.ParseUint(k, 10, 64)
		})
	manifestCacheSize := util.GetEnvAs[uint64](
		manifestCacheSizeEnvVar, DefaultCacheSize, func(k string) (uint64, error) {
			return strconv.ParseUint(k, 10, 64)
		})

	// Used by the requester node to map user provided docker image identifiers
	// to a version of the identifier with a digest.
	DockerTagCache, _ = basic.NewCache[string](
		basic.WithCleanupFrequency(tagCacheDuration),
		basic.WithMaxCost(tagCacheSize),
	)

	// Used by compute nodes to map requester provided image identifiers (with
	// digest) to
	DockerManifestCache, _ = basic.NewCache[ImageManifest](
		basic.WithCleanupFrequency(manifestCacheDuration),
		basic.WithMaxCost(manifestCacheSize),
	)
}
