package compute

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	executor_util "github.com/bacalhau-project/bacalhau/pkg/executor/util"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
)

func ExecutorProviders(nodeID types.NodeID, cfg types.ExecutorProvidersConfig) (executor.ExecutorProvider, error) {
	pr, err := executor_util.NewStandardExecutorProvider(
		executor_util.StandardExecutorOptions{
			DockerID: fmt.Sprintf("bacalhau-%s", nodeID),
		},
	)
	if err != nil {
		return nil, err
	}
	return provider.NewConfiguredProvider(pr, cfg.Disabled), err
}
