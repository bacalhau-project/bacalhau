package semantic

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type ProviderInstalledStrategy[K model.ProviderKey, P model.Providable] struct {
	provider model.Provider[K, P]
	getter   func(*model.Job) []K
}

func NewProviderInstalledStrategy[K model.ProviderKey, P model.Providable](
	provider model.Provider[K, P],
	getter func(*model.Job) K,
) *ProviderInstalledStrategy[K, P] {
	return &ProviderInstalledStrategy[K, P]{
		provider: provider,
		getter:   func(j *model.Job) []K { return []K{getter(j)} },
	}
}

func NewProviderInstalledArrayStrategy[K model.ProviderKey, P model.Providable](
	provider model.Provider[K, P],
	getter func(*model.Job) []K,
) *ProviderInstalledStrategy[K, P] {
	return &ProviderInstalledStrategy[K, P]{
		provider: provider,
		getter:   getter,
	}
}

func (s *ProviderInstalledStrategy[K, P]) ShouldBid(
	ctx context.Context,
	request bidstrategy.BidStrategyRequest,
) (resp bidstrategy.BidStrategyResponse, err error) {
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
