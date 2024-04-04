package compute

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/executor"
	executor_util "github.com/bacalhau-project/bacalhau/pkg/executor/util"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/node"
)

func ExecutorProviders(cfg node.ComputeConfig) (executor.ExecutorProvider, error) {
	pr, err := executor_util.NewStandardExecutorProvider(
		executor_util.StandardExecutorOptions{
			DockerID: fmt.Sprintf("bacalhau-%s", cfg.NodeID),
		},
	)
	if err != nil {
		return nil, err
	}
	return provider.NewConfiguredProvider(pr, cfg.DisabledFeatures.Engines), err
}
