package semantic

import (
	"context"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
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
		getter: func(j *models.Job) []string {
			// handle optional specs, such as publisher
			key := strings.TrimSpace(getter(j))
			if validate.IsBlank(key) {
				return []string{}
			}
			return []string{key}
		},
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
	keys := s.getter(&request.Job)
	for _, key := range keys {
		supportsKey := s.provider.Has(ctx, key)
		if !supportsKey {
			return bidstrategy.NewBidResponse(false, "support %q", key), nil
		}
	}

	return bidstrategy.NewBidResponse(true, "support %v", keys), nil
}
