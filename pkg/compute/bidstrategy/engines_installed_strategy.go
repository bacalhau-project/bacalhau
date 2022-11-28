package bidstrategy

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
)

type EnginesInstalledStrategyParams struct {
	Executors executor.ExecutorProvider
	Verifiers verifier.VerifierProvider
}

type EnginesInstalledStrategy struct {
	executors executor.ExecutorProvider
	verifiers verifier.VerifierProvider
}

func NewEnginesInstalledStrategy(params EnginesInstalledStrategyParams) *EnginesInstalledStrategy {
	return &EnginesInstalledStrategy{
		executors: params.Executors,
		verifiers: params.Verifiers,
	}
}

func (s *EnginesInstalledStrategy) ShouldBid(ctx context.Context, request BidStrategyRequest) (BidStrategyResponse, error) {
	// skip bidding if we don't have the executor and verifier for the job spec
	if !s.executors.HasExecutor(ctx, request.Job.Spec.Engine) {
		return BidStrategyResponse{
			ShouldBid: false,
			Reason:    fmt.Sprintf("executor %s not installed", request.Job.Spec.Engine),
		}, nil
	}

	if !s.verifiers.HasVerifier(ctx, request.Job.Spec.Verifier) {
		return BidStrategyResponse{
			ShouldBid: false,
			Reason:    fmt.Sprintf("verifier %s not installed", request.Job.Spec.Verifier),
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
