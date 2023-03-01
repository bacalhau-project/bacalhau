package bidstrategy

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type providerInstalledStrategy[K model.ProviderKey, P model.Providable] struct {
	provider model.Provider[K, P]
	getter   func(*model.Job) []K
}

func NewProviderInstalledStrategy[K model.ProviderKey, P model.Providable](
	provider model.Provider[K, P],
	getter func(*model.Job) K,
) BidStrategy {
	return &providerInstalledStrategy[K, P]{
		provider: provider,
		getter:   func(j *model.Job) []K { return []K{getter(j)} },
	}
}

func NewProviderInstalledArrayStrategy[K model.ProviderKey, P model.Providable](
	provider model.Provider[K, P],
	getter func(*model.Job) []K,
) BidStrategy {
	return &providerInstalledStrategy[K, P]{
		provider: provider,
		getter:   getter,
	}
}

func (s *providerInstalledStrategy[K, P]) ShouldBid(
	ctx context.Context,
	request BidStrategyRequest,
) (resp BidStrategyResponse, err error) {
	resp.ShouldBid = true
	for _, key := range s.getter(&request.Job) {
		resp.ShouldBid = s.provider.Has(ctx, key)
		resp.Reason = fmt.Sprintf("%s installed: %t", key, resp.ShouldBid)
		if !resp.ShouldBid {
			return
		}
	}

	return
}

func (s *providerInstalledStrategy[K, P]) ShouldBidBasedOnUsage(
	_ context.Context, _ BidStrategyRequest, _ model.ResourceUsageData) (BidStrategyResponse, error) {
	return NewShouldBidResponse(), nil
}
