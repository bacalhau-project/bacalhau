package semantic

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

type InputLocalityStrategyParams struct {
	Locality models.JobSelectionDataLocality
	Storages storage.StorageProvider
}

type InputLocalityStrategy struct {
	locality models.JobSelectionDataLocality
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

const (
	anywhereReason = "download data that is not available locally"
	nonLocalReason = "have all input data locally (%d inputs require downloading)"
)

func (s *InputLocalityStrategy) ShouldBid(
	ctx context.Context,
	request bidstrategy.BidStrategyRequest,
) (bidstrategy.BidStrategyResponse, error) {
	// if we have an "anywhere" policy for the data then we accept the job
	if s.locality == models.Anywhere {
		return bidstrategy.NewBidResponse(true, anywhereReason), nil
	}

	inputSources := request.Job.Task().InputSources
	foundInputs := 0
	for _, input := range inputSources {
		// see if the storage engine reports that we have the resource locally
		strg, err := s.storages.Get(ctx, input.Source.Type)
		if err != nil {
			return bidstrategy.BidStrategyResponse{}, err
		}
		hasStorage, err := strg.HasStorageLocally(ctx, *input)
		if err != nil {
			return bidstrategy.BidStrategyResponse{},
				fmt.Errorf("InputLocalityStrategy: failed to check for storage resource locality: %w", err)
		}
		if hasStorage {
			foundInputs++
		}
	}

	return bidstrategy.NewBidResponse(foundInputs >= len(inputSources), nonLocalReason, len(inputSources)-foundInputs), nil
}
