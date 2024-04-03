package provider

import (
	"fmt"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/executor/docker"
	"github.com/bacalhau-project/bacalhau/pkg/executor/wasm"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func Executor(cfg map[string][]byte) (executor.ExecutorProvider, error) {
	var (
		provided = make(map[string]executor.Executor)
		err      error
	)
	for name, config := range cfg {
		switch strings.ToLower(name) {
		case models.EngineDocker:
			provided[name], err = DockerEngine(config)
		case models.EngineWasm:
			provided[name], err = WasmEngine(config)
		default:
			return nil, fmt.Errorf("unknown executor provider: %s", name)
		}
		if err != nil {
			return nil, fmt.Errorf("registering %s executor: %w", name, err)
		}
	}
	return provider.NewMappedProvider(provided), nil
}

func DockerEngine(cfg []byte) (*docker.Executor, error) {
	panic("TODO")
	// return docker.NewExecutor(context.TODO(), nil)
}

func WasmEngine(cfg []byte) (*wasm.Executor, error) {
	return wasm.NewExecutor()
}
