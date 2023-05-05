package semantic

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type InputLocalityStrategyParams struct {
	Locality  model.JobSelectionDataLocality
	Executors executor.ExecutorProvider
}

type InputLocalityStrategy struct {
	locality  model.JobSelectionDataLocality
	executors executor.ExecutorProvider
}

// Compile-time check of interface implementation
var _ bidstrategy.SemanticBidStrategy = (*InputLocalityStrategy)(nil)

func NewInputLocalityStrategy(params InputLocalityStrategyParams) *InputLocalityStrategy {
	return &InputLocalityStrategy{
		locality:  params.Locality,
		executors: params.Executors,
	}
}

func (s *InputLocalityStrategy) ShouldBid(
	ctx context.Context,
	request bidstrategy.BidStrategyRequest,
) (bidstrategy.BidStrategyResponse, error) {
	// if we have an "anywhere" policy for the data then we accept the job
	if s.locality == model.Anywhere {
		return bidstrategy.NewShouldBidResponse(), nil
	}

	// otherwise we are checking that all the named inputs in the job
	// are local to us
	e, err := s.executors.Get(ctx, request.Job.Spec.Engine)
	if err != nil {
		return bidstrategy.BidStrategyResponse{}, fmt.Errorf("InputLocalityStrategy: failed to get executor %s: %w", request.Job.Spec.Engine, err)
	}

	foundInputs := 0

	for _, input := range request.Job.Spec.Inputs {
		// see if the storage engine reports that we have the resource locally
		hasStorage, err := e.HasStorageLocally(ctx, input)
		if err != nil {
			return bidstrategy.BidStrategyResponse{}, fmt.Errorf("InputLocalityStrategy: failed to check for storage resource locality: %w", err)
		}
		if hasStorage {
			foundInputs++
		}
	}

	if foundInputs >= len(request.Job.Spec.Inputs) {
		return bidstrategy.NewShouldBidResponse(), nil
	}
	return bidstrategy.BidStrategyResponse{ShouldBid: false, Reason: "not all inputs are local"}, nil
}
