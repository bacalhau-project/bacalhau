//go:build unit || !integration

package semantic_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/storage/noop"
)

var (
	OneStorageSpec []*models.InputSource = []*models.InputSource{
		{
			Source: models.NewSpecConfig(models.StorageSourceIPFS).WithParam("CID", "volume-id"),
			Target: "target",
		},
	}
)

var (
	EmptySpec      *models.Task
	SpecWithInputs *models.Task
	SpecWithWasm   *models.Task
)

func init() {
	EmptySpec = &models.Task{}
	SpecWithInputs = &models.Task{InputSources: OneStorageSpec}
	SpecWithWasm = mock.Task()
	SpecWithWasm.Engine = models.NewSpecConfig(models.EngineWasm).WithParam(model.EngineKeyEntryModuleWasm, OneStorageSpec)
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
		{"uninstalled storage/Wasm", SpecWithWasm, false, require.False},
		{"installed storage/Wasm", SpecWithWasm, true, require.True},
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
