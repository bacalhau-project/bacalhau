package semantic

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/compute"
)

type EnvResolverStrategyParams struct {
	Resolver compute.EnvVarResolver
}

type EnvResolverStrategy struct {
	resolver compute.EnvVarResolver
}

// Compile-time check of interface implementation
var _ bidstrategy.SemanticBidStrategy = (*EnvResolverStrategy)(nil)

func NewEnvResolverStrategy(params EnvResolverStrategyParams) *EnvResolverStrategy {
	return &EnvResolverStrategy{
		resolver: params.Resolver,
	}
}

const (
	noEnvVarsReason  = "accept jobs without environment variables"
	canResolveReason = "resolve all required environment variables"
)

func (s *EnvResolverStrategy) ShouldBid(
	ctx context.Context,
	request bidstrategy.BidStrategyRequest,
) (bidstrategy.BidStrategyResponse, error) {
	// If no env vars are requested, we can bid
	if len(request.Job.Task().Env) == 0 {
		return bidstrategy.NewBidResponse(true, noEnvVarsReason), nil
	}

	// Check if we can resolve all environment variables
	for name, value := range request.Job.Task().Env {
		if err := s.resolver.Validate(name, string(value)); err != nil {
			return bidstrategy.NewBidResponse(false, fmt.Sprintf("resolve environment variable %s: %v", name, err)), nil
		}
	}

	return bidstrategy.NewBidResponse(true, canResolveReason), nil
}
