package docker

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	dockerspec "github.com/bacalhau-project/bacalhau/pkg/model/engine/docker"
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

	engineSpec, err := dockerspec.Decode(request.Job.Spec.EngineSpec)
	if err != nil {
		return bidstrategy.BidStrategyResponse{
			ShouldBid:  false,
			ShouldWait: false,
			Reason:     err.Error(),
		}, err
	}

	supported, err := s.client.SupportedPlatforms(ctx)
	if err != nil {
		return bidstrategy.BidStrategyResponse{
			ShouldBid:  false,
			ShouldWait: false,
			Reason:     err.Error(),
		}, err
	}

	platforms, err := s.client.ImagePlatforms(ctx, engineSpec.Image, config.GetDockerCredentials())
	if err != nil {
		return bidstrategy.BidStrategyResponse{
			ShouldBid:  false,
			ShouldWait: false,
			Reason:     err.Error(),
		}, err
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
