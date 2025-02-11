//go:build unit || !integration

package wasm

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/storage/inline"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/bacalhau-project/bacalhau/testdata/wasm/dynamic"
	"github.com/bacalhau-project/bacalhau/testdata/wasm/easter"
	"github.com/bacalhau-project/bacalhau/testdata/wasm/noop"
)

func prepareModule(t *testing.T, alias string, program []byte) storage.PreparedStorage {
	store := inline.NewStorage()
	spec := store.StoreBytes(program)
	inputSource := models.InputSource{Source: &spec, Alias: alias, Target: "/" + uuid.NewString()}
	preparedVolume, err := store.PrepareStorage(context.Background(), t.TempDir(), mock.Execution(), inputSource)
	require.NoError(t, err)

	return storage.PreparedStorage{
		InputSource: inputSource,
		Volume:      preparedVolume,
	}
}

func TestModuleLoading(t *testing.T) {
	logger.ConfigureTestLogging(t)

	testCases := []struct {
		name          string
		entryModule   storage.PreparedStorage
		importModules []storage.PreparedStorage
		errorChecker  func(t require.TestingT, err error, _ ...any)
		moduleChecker func(t require.TestingT, object any, _ ...any)
	}{
		{
			name:          "simple load of single entry module",
			entryModule:   prepareModule(t, "", noop.Program()),
			importModules: []storage.PreparedStorage{},
			errorChecker:  require.NoError,
			moduleChecker: require.NotNil,
		},
		{
			name:        "does not attempt to load every input source as a WASM module",
			entryModule: prepareModule(t, "", noop.Program()),
			importModules: []storage.PreparedStorage{
				prepareModule(t, "something", []byte{0x0, 0x1, 0x2}),
			},
			errorChecker:  require.NoError,
			moduleChecker: require.NotNil,
		},
		{
			name:          "fails to load a module if it has a dependency that is not given",
			entryModule:   prepareModule(t, "", dynamic.Program()),
			importModules: []storage.PreparedStorage{},
			errorChecker:  require.Error,
			moduleChecker: require.Nil,
		},
		{
			name:        "loads module with a dependency if it is specified as an input source with the correct alias",
			entryModule: prepareModule(t, "", dynamic.Program()),
			importModules: []storage.PreparedStorage{
				// The alias being input.wasm is not relevant for this test – this is
				// just the "module name" that the `dynamic` program is looking
				// for in its WASM import header.
				prepareModule(t, "input.wasm", easter.Program()),
			},
			errorChecker:  require.NoError,
			moduleChecker: require.NotNil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			runtime := wazero.NewRuntime(ctx)
			config := wazero.NewModuleConfig()

			loader := NewModuleLoader(runtime, config, append(testCase.importModules, testCase.entryModule)...)
			require.NotNil(t, loader)

			module, err := loader.InstantiateRemoteModule(ctx, testCase.entryModule)
			testCase.errorChecker(t, err)
			testCase.moduleChecker(t, module)
		})
	}
}
