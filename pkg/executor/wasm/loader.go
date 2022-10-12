package wasm

import (
	"context"
	"os"

	"github.com/tetratelabs/wazero"
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
