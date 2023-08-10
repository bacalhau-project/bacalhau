package semantic

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type ProviderInstalledStrategy[P provider.Providable] struct {
	provider provider.Provider[P]
	getter   func(*models.Job) []string
}

func NewProviderInstalledStrategy[P provider.Providable](
	provider provider.Provider[P],
	getter func(*models.Job) string,
) *ProviderInstalledStrategy[P] {
	return &ProviderInstalledStrategy[P]{
		provider: provider,
		getter:   func(j *models.Job) []string { return []string{getter(j)} },
	}
}

func NewProviderInstalledArrayStrategy[P provider.Providable](
	provider provider.Provider[P],
	getter func(*models.Job) []string,
) *ProviderInstalledStrategy[P] {
	return &ProviderInstalledStrategy[P]{
		provider: provider,
		getter:   getter,
	}
}

func (s *ProviderInstalledStrategy[P]) ShouldBid(
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
