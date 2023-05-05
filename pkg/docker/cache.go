package docker

import (
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/cache"
	"github.com/bacalhau-project/bacalhau/pkg/cache/basic"
)

//nolint:unused
var DockerTagCache cache.Cache[string]

func init() { //nolint:gochecknoinits
	DockerTagCache, _ = basic.NewCache[string](
		basic.WithCleanupFrequency(time.Duration(1)*time.Hour),
		basic.WithMaxCost(1000), //nolint:gomnd
	)
}
