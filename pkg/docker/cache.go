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
const DefaultCacheDuration = 1 * time.Hour
const DefaultTagCacheFrequency = DefaultCacheDuration
const DefaultManifestCacheFrequency = DefaultCacheDuration

const tagCacheSizeEnvVar = "DOCKER_TAG_CACHE_SIZE"
const tagCacheDurationEnvVar = "DOCKER_TAG_CACHE_DURATION"
const tagCacheCheckFrequencyEnvVar = "DOCKER_TAG_CACHE_FREQUENCY"

const manifestCacheSizeEnvVar = "DOCKER_MANIFEST_CACHE_SIZE"
const manifestCacheDurationEnvVar = "DOCKER_MANIFEST_CACHE_DURATION"
const manifestCacheCheckFrequencyEnvVar = "DOCKER_MANIFEST_CACHE_FREQUENCY"

func init() { //nolint:gochecknoinits
	DockerTagCache = InitTagCache()
	DockerManifestCache = InitManifestCache()
}

func InitTagCache() cache.Cache[string] {
	tagCacheDuration := util.GetEnvAs[time.Duration](
		tagCacheDurationEnvVar, DefaultCacheDuration, time.ParseDuration,
	)
	tagCacheFrequency := util.GetEnvAs[time.Duration](
		tagCacheCheckFrequencyEnvVar, DefaultTagCacheFrequency, time.ParseDuration,
	)
	tagCacheSize := util.GetEnvAs[uint64](
		tagCacheSizeEnvVar, DefaultCacheSize, func(k string) (uint64, error) {
			return strconv.ParseUint(k, 10, 64)
		})

	// Used by the requester node to map user provided docker image identifiers
	// to a version of the identifier with a digest.
	c, _ := basic.NewCache[string](
		basic.WithCleanupFrequency(tagCacheFrequency),
		basic.WithMaxCost(tagCacheSize),
		basic.WithTTL(tagCacheDuration),
	)
	return c
}

func InitManifestCache() cache.Cache[ImageManifest] {
	manifestCacheDuration := util.GetEnvAs[time.Duration](
		manifestCacheDurationEnvVar, DefaultCacheDuration, time.ParseDuration,
	)
	manifestCacheFrequency := util.GetEnvAs[time.Duration](
		manifestCacheCheckFrequencyEnvVar, DefaultManifestCacheFrequency, time.ParseDuration,
	)
	manifestCacheSize := util.GetEnvAs[uint64](
		manifestCacheSizeEnvVar, DefaultCacheSize, func(k string) (uint64, error) {
			return strconv.ParseUint(k, 10, 64)
		})

	// Used by compute nodes to map requester provided image identifiers (with
	// digest) to
	c, _ := basic.NewCache[ImageManifest](
		basic.WithCleanupFrequency(manifestCacheFrequency),
		basic.WithMaxCost(manifestCacheSize),
		basic.WithTTL(manifestCacheDuration),
	)
	return c
}
