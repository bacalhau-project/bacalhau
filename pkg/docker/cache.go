package docker

import (
	"github.com/bacalhau-project/bacalhau/pkg/cache"
	"github.com/bacalhau-project/bacalhau/pkg/cache/basic"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

func NewManifestCache(cfg types.DockerManifestCache) cache.Cache[ImageManifest] {
	// Used by compute nodes to map requester provided image identifiers (with
	// digest) to
	c, _ := basic.NewCache[ImageManifest](
		basic.WithCleanupFrequency(cfg.Refresh.AsTimeDuration()),
		basic.WithMaxCost(cfg.Size),
		basic.WithTTL(cfg.TTL.AsTimeDuration()),
	)
	return c
}
