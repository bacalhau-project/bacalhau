package semantic

import (
	"context"

	"go.uber.org/multierr"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/cache"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/rs/zerolog/log"
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
	if request.Job.Spec.Engine != model.EngineDocker {
		return bidstrategy.NewShouldBidResponse(), nil
	}

	supported, serr := s.client.SupportedPlatforms(ctx)

	var ierr error = nil
	var manifest docker.ImageManifest

	manifest, found := (*ManifestCache).Get(request.Job.Spec.Docker.Image)
	if !found {
		log.Ctx(ctx).Debug().Str("Image", request.Job.Spec.Docker.Image).Msg("Image not found in manifest cache")

		var m *docker.ImageManifest
		m, ierr = s.client.ImageDistribution(ctx, request.Job.Spec.Docker.Image, config.GetDockerCredentials())
		if m != nil {
			manifest = *m
		}
	} else {
		log.Ctx(ctx).Debug().Str("Image", request.Job.Spec.Docker.Image).Msg("Image found in manifest cache")
	}

	err := multierr.Combine(serr, ierr)
	if err != nil {
		return bidstrategy.BidStrategyResponse{
			ShouldBid: false,
			Reason:    err.Error(),
		}, nil
	}

	// Cache the platform info for this image tag for a day. We could cache
	// for longer but we only have in-memory caches with time-based eviction.
	// TODO: Once we have an LRU cache we can use that instead and not worry
	// about managing eviction. In the meantime we get this through calling
	// Set even when don't have to, to reset the expiry time.
	err = (*ManifestCache).Set(
		request.Job.Spec.Docker.Image, manifest, 1, oneDayInSeconds,
	) //nolint:gomnd
	if err != nil {
		// Log the error but continue as it is not serious enough to stop
		// processing
		log.Ctx(ctx).Warn().
			Str("Image", request.Job.Spec.Docker.Image).
			Str("Error", err.Error()).
			Msg("Failed to save to manifest cache")
	}

	for _, canRun := range supported {
		for _, imageHas := range manifest.Platforms {
			if canRun.OS == imageHas.OS && canRun.Architecture == imageHas.Architecture {
				return bidstrategy.NewShouldBidResponse(), nil
			}
		}
	}

	return bidstrategy.BidStrategyResponse{
		ShouldBid: false,
		Reason:    "Node does not support any of the published image platforms",
	}, nil
}
