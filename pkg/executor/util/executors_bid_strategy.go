package util

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type bidStrategyFromExecutor struct {
	provider executor.ExecutorProvider
}

func NewExecutorSpecificBidStrategy(provider executor.ExecutorProvider) bidstrategy.BidStrategy {
	return bidstrategy.NewChainedBidStrategy(
		bidstrategy.WithSemantics(
			semantic.NewProviderInstalledStrategy[model.Engine, executor.Executor](
				provider,
				func(j *model.Job) model.Engine {
					return j.Spec.EngineSpec.Engine()
				},
			),
			&bidStrategyFromExecutor{
				provider: provider,
			},
		),
		bidstrategy.WithResources(
			&bidStrategyFromExecutor{
				provider: provider,
			},
		),
	)
}

// ShouldBid implements bidstrategy.BidStrategy
func (p *bidStrategyFromExecutor) ShouldBid(
	ctx context.Context,
	request bidstrategy.BidStrategyRequest,
) (bidstrategy.BidStrategyResponse, error) {
	e, err := p.provider.Get(ctx, request.Job.Spec.EngineSpec.Engine())
	if err != nil {
		return bidstrategy.BidStrategyResponse{}, err
	}

	return e.ShouldBid(ctx, request)
}

// ShouldBidBasedOnUsage implements bidstrategy.BidStrategy
func (p *bidStrategyFromExecutor) ShouldBidBasedOnUsage(
	ctx context.Context,
	request bidstrategy.BidStrategyRequest,
	resourceUsage model.ResourceUsageData,
) (bidstrategy.BidStrategyResponse, error) {
	e, err := p.provider.Get(ctx, request.Job.Spec.EngineSpec.Engine())
	if err != nil {
		return bidstrategy.BidStrategyResponse{}, err
	}

	return e.ShouldBidBasedOnUsage(ctx, request, resourceUsage)
}

var _ bidstrategy.ResourceBidStrategy = (*bidStrategyFromExecutor)(nil)
var _ bidstrategy.SemanticBidStrategy = (*bidStrategyFromExecutor)(nil)
