package docker

import (
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/cache"
)

//nolint:unused
var DockerTagCache cache.Cache[string]

func init() { //nolint:gochecknoinits
	tagCacheOptions := cache.NewCacheOptions(1000, time.Duration(1)*time.Hour) //nolint:gomnd
	DockerTagCache, _ = cache.NewBasicCache[string](tagCacheOptions)
}
