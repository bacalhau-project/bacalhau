package docker

import (
	"context"

	"go.uber.org/multierr"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/executor/docker/spec"
	"github.com/bacalhau-project/bacalhau/pkg/model"
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

	// FIXME: this looks like a bug, we should return false as this isn't a docker spec.
	if request.Job.Spec.EngineSpec.Type != model.EngineDocker {
		return bidstrategy.NewShouldBidResponse(), nil
	}

	engineSpec, err := spec.AsJobSpecDocker(request.Job.Spec.EngineSpec)
	if err != nil {
		return bidstrategy.BidStrategyResponse{
			ShouldBid:  false,
			ShouldWait: false,
			Reason:     err.Error(),
		}, err
	}

	supported, serr := s.client.SupportedPlatforms(ctx)
	platforms, ierr := s.client.ImagePlatforms(ctx, engineSpec.Image, config.GetDockerCredentials())
	err = multierr.Combine(serr, ierr)
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
