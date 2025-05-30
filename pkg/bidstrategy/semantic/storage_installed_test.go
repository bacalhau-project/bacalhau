//go:build unit || !integration

package semantic_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/storage/noop"
)

var (
	OneStorageSpec = []*models.InputSource{
		{
			Source: models.NewSpecConfig(models.StorageSourceLocal).WithParam("SourcePath", "/dummy/path"),
			Target: "target",
		},
	}
)

var (
	EmptySpec      *models.Task
	SpecWithInputs *models.Task
)

func init() {
	EmptySpec = mock.Task()
	EmptySpec.InputSources = make([]*models.InputSource, 0)

	SpecWithInputs = mock.Task()
	SpecWithInputs.InputSources = OneStorageSpec
}

func TestStorageBidStrategy(t *testing.T) {
	testCases := []struct {
		name      string
		spec      *models.Task
		installed bool
		check     func(require.TestingT, bool, ...any)
	}{
		{"no storage", EmptySpec, true, require.True},
		{"no storage with nothing installed", EmptySpec, false, require.True},
		{"uninstalled storage/Inputs", SpecWithInputs, false, require.False},
		{"installed storage/Inputs", SpecWithInputs, true, require.True},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			noop_storage := noop.NewNoopStorageWithConfig(noop.StorageConfig{
				ExternalHooks: noop.StorageConfigExternalHooks{
					IsInstalled: func(ctx context.Context) (bool, error) {
						return testCase.installed, nil
					},
				},
			})
			provider := provider.NewNoopProvider[storage.Storage](noop_storage)
			strategy := semantic.NewStorageInstalledBidStrategy(provider)

			result, err := strategy.ShouldBid(context.Background(), bidstrategy.BidStrategyRequest{
				Job: models.Job{
					Tasks: []*models.Task{testCase.spec},
				},
			})
			require.NoError(t, err)
			testCase.check(t, result.ShouldBid, fmt.Sprintf("Reason: %q", result.Reason))
		})
	}
}
