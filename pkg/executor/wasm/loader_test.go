//go:build unit || !integration

package wasm

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"

	"github.com/bacalhau-project/bacalhau/pkg/executor/wasm/util/filefs"
	"github.com/bacalhau-project/bacalhau/pkg/executor/wasm/util/mountfs"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/testdata/wasm/dynamic"
	"github.com/bacalhau-project/bacalhau/testdata/wasm/easter"
	"github.com/bacalhau-project/bacalhau/testdata/wasm/noop"
)

func TestModuleLoading(t *testing.T) {
	logger.ConfigureTestLogging(t)

	// Create a single runtime for all test cases
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testCases := []struct {
		name          string
		entryModule   []byte
		importModules map[string][]byte
		errorChecker  func(t require.TestingT, err error, _ ...any)
		moduleChecker func(t require.TestingT, object any, _ ...any)
	}{
		{
			name:          "simple load of single entry module",
			entryModule:   noop.Program(),
			importModules: map[string][]byte{},
			errorChecker:  require.NoError,
			moduleChecker: require.NotNil,
		},
		{
			name:        "does not attempt to load every input source as a WASM module",
			entryModule: noop.Program(),
			importModules: map[string][]byte{
				"invalid": {0x0, 0x1, 0x2},
			},
			errorChecker:  require.NoError,
			moduleChecker: require.NotNil,
		},
		{
			name:          "fails to load a module if it has a dependency that is not given",
			entryModule:   dynamic.Program(),
			importModules: map[string][]byte{},
			errorChecker:  require.Error,
			moduleChecker: require.Nil,
		},
		{
			name:        "loads module with a dependency if it is specified as an input source with the correct alias",
			entryModule: dynamic.Program(),
			importModules: map[string][]byte{
				// The name being input.wasm is not relevant for this test – this is
				// just the "module name" that the `dynamic` program is looking
				// for in its WASM import header.
				"input.wasm": easter.Program(),
			},
			errorChecker:  require.NoError,
			moduleChecker: require.NotNil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Create a fresh filesystem for this test case
			testFS := createTestFS(t, testCase.entryModule, testCase.importModules)

			// Create a fresh runtime for this test case
			runtime := wazero.NewRuntime(ctx)
			config := wazero.NewModuleConfig()

			// Create a fresh loader for this test case
			loader := NewModuleLoader(runtime, config, testFS)
			require.NotNil(t, loader)

			// Load the entry module
			module, err := loader.InstantiateModule(ctx, "main.wasm")
			testCase.errorChecker(t, err)
			testCase.moduleChecker(t, module)
		})
	}
}

func createTestFS(t *testing.T, entryModule []byte, importModules map[string][]byte) fs.FS {
	// Create a temporary directory for our test filesystem
	tmpDir := t.TempDir()

	// Create a mountable filesystem
	rootFs := mountfs.New()

	// Copy entry module to a temporary file and mount it
	entryFile := filepath.Join(tmpDir, "main.wasm")
	err := os.WriteFile(entryFile, entryModule, 0644)
	require.NoError(t, err)
	err = rootFs.Mount("main.wasm", filefs.New(entryFile))
	require.NoError(t, err)

	// Copy import modules to temporary files and mount them
	for target, data := range importModules {
		importFile := filepath.Join(tmpDir, filepath.Base(target))
		err := os.WriteFile(importFile, data, 0644)
		require.NoError(t, err)
		err = rootFs.Mount(target, filefs.New(importFile))
		require.NoError(t, err)
	}

	return rootFs
}
