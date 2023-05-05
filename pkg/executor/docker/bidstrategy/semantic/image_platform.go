package semantic

import (
	"context"

	"go.uber.org/multierr"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

var _ bidstrategy.SemanticBidStrategy = (*ImagePlatformBidStrategy)(nil)

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
	platforms, ierr := s.client.ImagePlatforms(ctx, request.Job.Spec.Docker.Image, config.GetDockerCredentials())
	err := multierr.Combine(serr, ierr)
	if err != nil {
		return bidstrategy.BidStrategyResponse{
			ShouldBid: false,
			Reason:    err.Error(),
		}, nil
	}

	for _, canRun := range supported {
		for _, imageHas := range platforms {
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
