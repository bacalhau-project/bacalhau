package nodefx

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/executor/docker"
	"github.com/bacalhau-project/bacalhau/pkg/executor/wasm"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func ExecutorProviders(cfg *ComputeConfig) (executor.ExecutorProvider, error) {
	var (
		provided = make(map[string]executor.Executor)
		err      error
	)

	c := cfg.Providers.Executor
	for name, config := range c {
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
	var ecfg docker.Config
	if err := json.Unmarshal(cfg, &ecfg); err != nil {
		return nil, err
	}
	return docker.NewExecutorFromConfig(ecfg)
}

func WasmEngine(cfg []byte) (*wasm.Executor, error) {
	return wasm.NewExecutor()
}
