package docker

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"go.uber.org/multierr"
)

func NewBidStrategy(client *docker.Client) bidstrategy.BidStrategy {
	return &imagePlatformBidStrategy{client: client}
}

type imagePlatformBidStrategy struct {
	client *docker.Client
}

// ShouldBid implements bidstrategy.BidStrategy
func (s *imagePlatformBidStrategy) ShouldBid(
	ctx context.Context,
	request bidstrategy.BidStrategyRequest,
) (bidstrategy.BidStrategyResponse, error) {
	if request.Job.Spec.Engine != model.EngineDocker {
		return bidstrategy.NewShouldBidResponse(), nil
	}

	supported, serr := s.client.SupportedPlatforms(ctx)
	platforms, ierr := s.client.ImagePlatforms(ctx, request.Job.Spec.Docker.Image)
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

// ShouldBidBasedOnUsage implements bidstrategy.BidStrategy
func (*imagePlatformBidStrategy) ShouldBidBasedOnUsage(
	ctx context.Context,
	request bidstrategy.BidStrategyRequest,
	resourceUsage model.ResourceUsageData,
) (bidstrategy.BidStrategyResponse, error) {
	return bidstrategy.NewShouldBidResponse(), nil
}

var _ bidstrategy.BidStrategy = (*imagePlatformBidStrategy)(nil)
