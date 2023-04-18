package util

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type bidStrategyFromExecutor struct {
	provider executor.ExecutorProvider
}

func NewExecutorSpecificBidStrategy(provider executor.ExecutorProvider) bidstrategy.BidStrategy {
	return bidstrategy.NewChainedBidStrategy(
		bidstrategy.NewProviderInstalledStrategy[model.Engine, executor.Executor](
			provider,
			func(j *model.Job) model.Engine { return j.Spec.EngineSpec.Type },
		),
		&bidStrategyFromExecutor{
			provider: provider,
		},
	)
}

// ShouldBid implements bidstrategy.BidStrategy
func (p *bidStrategyFromExecutor) ShouldBid(
	ctx context.Context,
	request bidstrategy.BidStrategyRequest,
) (bidstrategy.BidStrategyResponse, error) {
	executor, err := p.provider.Get(ctx, request.Job.Spec.EngineSpec.Type)
	if err != nil {
		return bidstrategy.BidStrategyResponse{}, err
	}

	strategy, err := executor.GetBidStrategy(ctx)
	if err != nil {
		return bidstrategy.BidStrategyResponse{}, err
	}

	return strategy.ShouldBid(ctx, request)
}

// ShouldBidBasedOnUsage implements bidstrategy.BidStrategy
func (p *bidStrategyFromExecutor) ShouldBidBasedOnUsage(
	ctx context.Context,
	request bidstrategy.BidStrategyRequest,
	resourceUsage model.ResourceUsageData,
) (bidstrategy.BidStrategyResponse, error) {
	executor, err := p.provider.Get(ctx, request.Job.Spec.EngineSpec.Type)
	if err != nil {
		return bidstrategy.BidStrategyResponse{}, err
	}

	strategy, err := executor.GetBidStrategy(ctx)
	if err != nil {
		return bidstrategy.BidStrategyResponse{}, err
	}

	return strategy.ShouldBidBasedOnUsage(ctx, request, resourceUsage)
}

var _ bidstrategy.BidStrategy = (*bidStrategyFromExecutor)(nil)
