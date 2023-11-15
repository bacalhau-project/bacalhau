package semantic

import (
	"context"

	"go.uber.org/multierr"

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

var ManifestCache *cache.Cache[docker.ImageManifest] = &docker.DockerManifestCache

func NewImagePlatformBidStrategy(client *docker.Client) *ImagePlatformBidStrategy {
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

	supported, serr := s.client.SupportedPlatforms(ctx)

	var ierr error = nil
	var manifest docker.ImageManifest

	dockerEngine, err := dockermodels.DecodeSpec(request.Job.Task().Engine)
	if err != nil {
		return bidstrategy.BidStrategyResponse{
			ShouldBid:  false,
			ShouldWait: false,
			Reason:     err.Error(),
		}, nil
	}

	manifest, found := (*ManifestCache).Get(dockerEngine.Image)
	if !found {
		log.Ctx(ctx).Debug().Str("Image", dockerEngine.Image).Msg("Image not found in manifest cache")

		var m *docker.ImageManifest
		m, ierr = s.client.ImageDistribution(ctx, dockerEngine.Image, config.GetDockerCredentials())
		if m != nil {
			manifest = *m
		}
		// Cache the platform info for this image tag for a day. We could cache
		// for longer but we only have in-memory caches with time-based eviction.
		// TODO: Once we have an LRU cache we can use that instead and not worry
		// about managing eviction. In the meantime we get this through calling
		// Set even when don't have to, to reset the expiry time.
		defer func() {
			err = (*ManifestCache).Set(
				dockerEngine.Image, manifest, 1, oneDayInSeconds,
			) //nolint:gomnd
			if err != nil {
				// Log the error but continue as it is not serious enough to stop
				// processing
				log.Ctx(ctx).Warn().
					Str("Image", dockerEngine.Image).
					Str("Error", err.Error()).
					Msg("Failed to save to manifest cache")
			}
		}()
	} else {
		log.Ctx(ctx).Debug().Str("Image", dockerEngine.Image).Msg("Image found in manifest cache")
	}

	errs := multierr.Combine(serr, ierr)
	if errs != nil {
		return bidstrategy.BidStrategyResponse{
			ShouldBid: false,
			Reason:    errs.Error(),
		}, nil
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
