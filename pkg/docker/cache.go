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

const DefaultCacheSize = uint64(1000)
const DefaultCacheDuration = time.Hour

const TagCacheSizeEnvVar = "DOCKER_TAG_CACHE_SIZE"
const TagCacheDurationEnvVar = "DOCKER_TAG_CACHE_DURATION"

func init() { //nolint:gochecknoinits
	duration := util.GetEnvAs[time.Duration](
		TagCacheDurationEnvVar, DefaultCacheDuration, time.ParseDuration,
	)
	size := util.GetEnvAs[uint64](
		TagCacheSizeEnvVar, DefaultCacheSize, func(k string) (uint64, error) {
			return strconv.ParseUint(k, 10, 64)
		})

	DockerTagCache, _ = basic.NewCache[string](
		basic.WithCleanupFrequency(duration),
		basic.WithMaxCost(size),
	)
}
