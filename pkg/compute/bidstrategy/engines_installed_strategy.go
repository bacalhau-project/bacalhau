package bidstrategy

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
)

type EnginesInstalledStrategyParams struct {
	Storages   storage.StorageProvider
	Executors  executor.ExecutorProvider
	Verifiers  verifier.VerifierProvider
	Publishers publisher.PublisherProvider
}

type EnginesInstalledStrategy struct {
	storages   storage.StorageProvider
	executors  executor.ExecutorProvider
	verifiers  verifier.VerifierProvider
	publishers publisher.PublisherProvider
}

func NewEnginesInstalledStrategy(params EnginesInstalledStrategyParams) *EnginesInstalledStrategy {
	return &EnginesInstalledStrategy{
		storages:   params.Storages,
		executors:  params.Executors,
		verifiers:  params.Verifiers,
		publishers: params.Publishers,
	}
}

func (s *EnginesInstalledStrategy) ShouldBid(ctx context.Context, request BidStrategyRequest) (BidStrategyResponse, error) {
	// skip bidding if we don't have the executor and verifier for the job spec
	for _, input := range request.Job.Spec.Inputs {
		if !s.storages.Has(ctx, input.StorageSource) {
			return BidStrategyResponse{
				ShouldBid: false,
				Reason:    fmt.Sprintf("storage %s not installed", input.StorageSource),
			}, nil
		}
	}

	if !s.executors.Has(ctx, request.Job.Spec.Engine) {
		return BidStrategyResponse{
			ShouldBid: false,
			Reason:    fmt.Sprintf("executor %s not installed", request.Job.Spec.Engine),
		}, nil
	}

	if !s.verifiers.Has(ctx, request.Job.Spec.Verifier) {
		return BidStrategyResponse{
			ShouldBid: false,
			Reason:    fmt.Sprintf("verifier %s not installed", request.Job.Spec.Verifier),
		}, nil
	}

	if !s.publishers.Has(ctx, request.Job.Spec.Publisher) {
		return BidStrategyResponse{
			ShouldBid: false,
			Reason:    fmt.Sprintf("publisher %s not installed", request.Job.Spec.Publisher),
		}, nil
	}

	return newShouldBidResponse(), nil
}

func (s *EnginesInstalledStrategy) ShouldBidBasedOnUsage(
	_ context.Context, _ BidStrategyRequest, _ model.ResourceUsageData) (BidStrategyResponse, error) {
	return newShouldBidResponse(), nil
}

// Compile-time check of interface implementation
var _ BidStrategy = (*EnginesInstalledStrategy)(nil)
