package semantic

import (
	"context"
	"sync"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	dockermodels "github.com/bacalhau-project/bacalhau/pkg/executor/docker/models"
	"github.com/bacalhau-project/bacalhau/pkg/models"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/cache"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
)

const oneDayInSeconds = int64(86400)

var _ bidstrategy.SemanticBidStrategy = (*ImagePlatformBidStrategy)(nil)

var ManifestCache cache.Cache[docker.ImageManifest]
var mu sync.Mutex

func NewImagePlatformBidStrategy(client *docker.Client, cfg types.DockerCacheConfig) *ImagePlatformBidStrategy {
	mu.Lock()
	// We will create the local reference to a manifest cache on demand,
	// ensuring that we lock access to the cache here to avoid race
	// conditions
	if ManifestCache == nil {
		ManifestCache = docker.NewManifestCache(cfg)
	}
	mu.Unlock()

	return &ImagePlatformBidStrategy{client: client}
}

type ImagePlatformBidStrategy struct {
	client *docker.Client
}

// ShouldBid implements semantic.SemanticBidStrategy
func (s *ImagePlatformBidStrategy) ShouldBid(
	ctx context.Context,
	request bidstrategy.BidStrategyRequest,
) (bidstrategy.BidStrategyResponse, error) {
	if request.Job.Task().Engine.Type != models.EngineDocker {
		return bidstrategy.NewBidResponse(true, "examine images for non-Docker jobs"), nil
	}

	supported, err := s.client.SupportedPlatforms(ctx)
	if err != nil {
		return bidstrategy.BidStrategyResponse{}, err
	}

	dockerEngine, err := dockermodels.DecodeSpec(request.Job.Task().Engine)
	if err != nil {
		return bidstrategy.BidStrategyResponse{}, err
	}

	manifest, found := ManifestCache.Get(dockerEngine.Image)
	if !found {
		log.Ctx(ctx).Debug().Str("Image", dockerEngine.Image).Msg("Image not found in manifest cache")

		creds := config.GetDockerCredentials()

		m, err := s.client.ImageDistribution(ctx, dockerEngine.Image, creds)
		if err != nil {
			return bidstrategy.BidStrategyResponse{}, err
		}

		if m != nil {
			manifest = *m
		}
		// Cache the platform info for this image tag for a day. We could cache
		// for longer but we only have in-memory caches with time-based eviction.
		// TODO: Once we have an LRU cache we can use that instead and not worry
		// about managing eviction. In the meantime we get this through calling
		// Set even when don't have to, to reset the expiry time.
		defer func() {
			err = ManifestCache.Set(
				dockerEngine.Image, manifest, 1, oneDayInSeconds,
			) //nolint:gomnd
			if err != nil {
				// Log the error but continue as it is not serious enough to stop
				// processing
				log.Ctx(ctx).Warn().
					Str("Image", dockerEngine.Image).
					Err(err).
					Msg("Failed to save to manifest cache")
			}
		}()
	} else {
		log.Ctx(ctx).Debug().Str("Image", dockerEngine.Image).Msg("Image found in manifest cache")
	}

	imageHasPlatforms := make([]string, 0, len(manifest.Platforms))
	for _, imageHas := range manifest.Platforms {
		imageHasPlatforms = append(imageHasPlatforms, imageHas.OS+"/"+imageHas.Architecture)
	}

	canRunPlatforms := make([]string, 0, len(supported))
	for _, canRun := range supported {
		canRunPlatforms = append(canRunPlatforms, canRun.OS+"/"+canRun.Architecture)
	}

	shouldBid := false
	for _, canRun := range supported {
		for _, imageHas := range manifest.Platforms {
			if canRun.OS == imageHas.OS && canRun.Architecture == imageHas.Architecture {
				shouldBid = true
			}
		}
	}

	const platformReason = "support the available image platforms %v (supports %v)"
	return bidstrategy.NewBidResponse(shouldBid, platformReason, imageHasPlatforms, canRunPlatforms), nil
}
