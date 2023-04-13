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
	if request.Job.Spec.EngineSpec.Type != model.EngineDocker {
		return bidstrategy.NewShouldBidResponse(), nil
	}

	dockerEngineSpec, err := spec.AsDockerSpec(request.Job.Spec.EngineSpec)
	if err != nil {
		// FIXME(forrest): this method (this specific impl) never returns an error and some callers don't check the error, instead they expect a response
		// containing an error. This method should either return an error and no response, or a response and no error
		// never both as it's not idiomatic go. The interface may need to be changed to adapt to this.
		// Gut check says the simplest solution is to remove the error type from the return and always include the error in the respones.

		// TODO(forrest): For now return both I guess?
		return bidstrategy.BidStrategyResponse{
			ShouldBid: false,
			Reason:    err.Error(),
		}, err
	}
	supported, serr := s.client.SupportedPlatforms(ctx)
	platforms, ierr := s.client.ImagePlatforms(ctx, dockerEngineSpec.Image, config.GetDockerCredentials())
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
