package semantic

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

type InputLocalityStrategyParams struct {
	Locality model.JobSelectionDataLocality
	Storages storage.StorageProvider
}

type InputLocalityStrategy struct {
	locality model.JobSelectionDataLocality
	storages storage.StorageProvider
}

// Compile-time check of interface implementation
var _ bidstrategy.SemanticBidStrategy = (*InputLocalityStrategy)(nil)

func NewInputLocalityStrategy(params InputLocalityStrategyParams) *InputLocalityStrategy {
	return &InputLocalityStrategy{
		locality: params.Locality,
		storages: params.Storages,
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

	foundInputs := 0
	for _, input := range request.Job.Spec.Inputs {
		// see if the storage engine reports that we have the resource locally
		strg, err := s.storages.Get(ctx, input.StorageSource)
		if err != nil {
			return bidstrategy.BidStrategyResponse{}, err
		}
		hasStorage, err := strg.HasStorageLocally(ctx, input)
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
