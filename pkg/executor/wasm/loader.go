package wasm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/rs/zerolog/log"
	"github.com/tetratelabs/wazero"
	"golang.org/x/exp/maps"
)

func LoadModule(ctx context.Context, runtime wazero.Runtime, path string) (wazero.CompiledModule, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	module, err := runtime.CompileModule(ctx, bytes)
	if err != nil {
		return nil, err
	}

	return module, nil
}

func LoadRemoteModule(
	ctx context.Context,
	runtime wazero.Runtime,
	provider storage.StorageProvider,
	spec model.StorageSpec,
) (wazero.CompiledModule, error) {
	volumes, err := storage.ParallelPrepareStorage(ctx, provider, []model.StorageSpec{spec})
	if err != nil {
		return nil, err
	}
	volume := maps.Values(volumes)[0]

	programPath := volume.Source
	info, err := os.Stat(programPath)
	if err != nil {
		return nil, err
	}

	// We expect the input to be a single WASM file. It is common however for
	// IPFS implementations to wrap files into a directory. So we make a special
	// case here – if the input is a single file in a directory, we will assume
	// this is the program file and load it.
	if info.IsDir() {
		files, err := os.ReadDir(programPath)
		if err != nil {
			return nil, err
		}

		if len(files) != 1 {
			return nil, fmt.Errorf("should be %d file in %s but there are %d", 1, programPath, len(files))
		}
		programPath = filepath.Join(programPath, files[0].Name())
	}

	log.Ctx(ctx).Debug().Msgf("Loading WASM module from %q", programPath)
	return LoadModule(ctx, runtime, programPath)
}
