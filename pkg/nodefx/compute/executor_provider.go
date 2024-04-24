package compute

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/executor/docker"
	"github.com/bacalhau-project/bacalhau/pkg/executor/wasm"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func ExecutorProviders(
	nodeID types.NodeID,
	exeCfg types.ExecutorProvidersConfig,
	dockerCredsCfg types.DockerCredentialsConfig,
	dockerCacheCfg types.DockerCacheConfig,
) (executor.ExecutorProvider, error) {
	dockerExecutor, err := docker.NewExecutor(fmt.Sprintf("bacalhau-%s", nodeID), dockerCredsCfg, dockerCacheCfg)
	if err != nil {
		return nil, err
	}

	wasmExecutor, err := wasm.NewExecutor()
	if err != nil {
		return nil, err
	}

	// TODO(forrest) [refactor]: use fx decorator to wrap these with tracing.
	ep := provider.NewMappedProvider(map[string]executor.Executor{
		models.EngineDocker: dockerExecutor,
		models.EngineWasm:   wasmExecutor,
	})
	return provider.NewConfiguredProvider[executor.Executor](ep, exeCfg.Disabled), nil
}
