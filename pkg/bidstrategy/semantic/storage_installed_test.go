//go:build unit || !integration

package semantic_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/storage/noop"
)

var (
	OneStorageSpec []model.StorageSpec = []model.StorageSpec{
		{StorageSource: model.StorageSourceIPFS},
	}
)

var (
	EmptySpec       = model.Spec{}
	SpecWithInputs  = model.Spec{Inputs: OneStorageSpec}
	SpecWithOutputs = model.Spec{Outputs: OneStorageSpec}
	SpecWithWasm    = model.Spec{Wasm: model.JobSpecWasm{EntryModule: OneStorageSpec[0]}}
)

func TestStorageBidStrategy(t *testing.T) {
	testCases := []struct {
		name      string
		spec      model.Spec
		installed bool
		check     func(require.TestingT, bool, ...any)
	}{
		{"no storage", EmptySpec, true, require.True},
		{"no storage with nothing installed", EmptySpec, false, require.True},
		{"uninstalled storage/Inputs", SpecWithInputs, false, require.False},
		{"installed storage/Inputs", SpecWithInputs, true, require.True},
		{"uninstalled storage/Outputs", SpecWithOutputs, false, require.False},
		{"installed storage/Outputs", SpecWithOutputs, true, require.True},
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
			provider := model.NewNoopProvider[model.StorageSourceType, storage.Storage](noop_storage)
			strategy := semantic.NewStorageInstalledBidStrategy(provider)

			result, err := strategy.ShouldBid(context.Background(), bidstrategy.BidStrategyRequest{
				Job: model.Job{
					Spec: testCase.spec,
				},
			})
			require.NoError(t, err)
			testCase.check(t, result.ShouldBid, fmt.Sprintf("Reason: %q", result.Reason))
		})
	}
}
