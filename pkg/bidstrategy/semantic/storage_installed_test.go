//go:build unit || !integration

package semantic_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/wasm"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/storage/noop"
)

var (
	OneStorageSpec spec.Storage
	EngineSpec     spec.Engine

	EmptySpec       model.Spec
	SpecWithInputs  model.Spec
	SpecWithOutputs model.Spec
	SpecWithWasm    model.Spec
)

func init() {
	c, err := cid.Decode("QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG")
	if err != nil {
		panic(err)
	}

	OneStorageSpec, err = (&ipfs.IPFSStorageSpec{CID: c}).AsSpec("TODO", "TODO")
	if err != nil {
		panic(err)
	}

	EngineSpec, err = (&wasm.WasmEngineSpec{EntryModule: &OneStorageSpec}).AsSpec()
	if err != nil {
		panic(err)
	}

	EmptySpec = model.Spec{}
	SpecWithInputs = model.Spec{Inputs: []spec.Storage{OneStorageSpec}}
	SpecWithOutputs = model.Spec{Outputs: []spec.Storage{OneStorageSpec}}
	SpecWithWasm = model.Spec{Engine: EngineSpec}
}

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
			provider := model.NewNoopProvider[cid.Cid, storage.Storage](noop_storage)
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
