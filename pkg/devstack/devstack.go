package devstack

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/config_legacy"
	"github.com/bacalhau-project/bacalhau/pkg/lib/network"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type DevStack struct {
	Nodes []*node.Node
}

//nolint:funlen,gocyclo
func Setup(
	ctx context.Context,
	cm *system.CleanupManager,
	opts ...ConfigOption,
) (*DevStack, error) {
	stackConfig, err := defaultDevStackConfig()
	if err != nil {
		return nil, fmt.Errorf("creating devstack defaults: %w", err)
	}
	for _, opt := range opts {
		opt(stackConfig)
	}

	if stackConfig.BasePath == "" {
		stackConfig.BasePath, err = os.MkdirTemp("", "bacalhau-devstack")
		if err != nil {
			return nil, fmt.Errorf("creating temporary directory: %w", err)
		}
	}

	if err = stackConfig.Validate(); err != nil {
		return nil, fmt.Errorf("validating devstask config: %w", err)
	}

	log.Ctx(ctx).Info().Object("Config", stackConfig).Msg("Starting Devstack")

	cfg := stackConfig.BacalhauConfig

	var nodes []*node.Node
	orchestratorAddrs := make([]string, 0)
	clusterPeersAddrs := make([]string, 0)

	totalNodeCount := stackConfig.NumberOfHybridNodes + stackConfig.NumberOfRequesterOnlyNodes + stackConfig.NumberOfComputeOnlyNodes
	requesterNodeCount := stackConfig.NumberOfHybridNodes + stackConfig.NumberOfRequesterOnlyNodes
	computeNodeCount := stackConfig.NumberOfHybridNodes + stackConfig.NumberOfComputeOnlyNodes

	if requesterNodeCount == 0 {
		return nil, fmt.Errorf("at least one requester node is required")
	}

	for i := 0; i < totalNodeCount; i++ {
		nodeID := fmt.Sprintf("node-%d", i)
		ctx = logger.ContextWithNodeIDLogger(ctx, nodeID)

		isRequesterNode := i < requesterNodeCount
		isComputeNode := (totalNodeCount - i) <= computeNodeCount
		log.Ctx(ctx).Debug().Msgf(`Creating Node #%d as {RequesterNode: %t, ComputeNode: %t}`, i+1, isRequesterNode, isComputeNode)

		// ////////////////////////////////////
		// Transport layer
		// ////////////////////////////////////
		if os.Getenv("PREDICTABLE_API_PORT") != "" {
			cfg.Orchestrator.Port = 4222 + i
		} else {
			if cfg.Orchestrator.Port, err = network.GetFreePort(); err != nil {
				return nil, errors.Wrap(err, "failed to get free port for nats server")
			}
		}

		if os.Getenv("PREDICTABLE_API_PORT") != "" {
			cfg.Orchestrator.Cluster.Port = 6222 + i
		} else {
			if cfg.Orchestrator.Cluster.Port, err = network.GetFreePort(); err != nil {
				return nil, errors.Wrap(err, "failed to get free port for nats cluster")
			}
		}

		if isComputeNode {
			cfg.Compute.Orchestrators = orchestratorAddrs
		}

		if isRequesterNode {
			cfg.Orchestrator.Cluster.Peers = clusterPeersAddrs
			cfg.Orchestrator.Cluster.Name = "devstack"
			orchestratorAddrs = append(orchestratorAddrs, fmt.Sprintf("127.0.0.1:%d", cfg.Orchestrator.Port))
		}

		// ////////////////////////////////////
		// port for API
		// ////////////////////////////////////
		if os.Getenv("PREDICTABLE_API_PORT") != "" {
			cfg.API.Port = 1234 + i
		} else {
			if cfg.API.Port, err = network.GetFreePort(); err != nil {
				return nil, errors.Wrap(err, "failed to get free port for API server")
			}
		}

		// ////////////////////////////////////
		// Create and Run Node
		// ////////////////////////////////////

		// here is where we can parse string based CLI stackConfig
		// into more meaningful model.FailureInjectionConfig values
		isBadComputeActor := (stackConfig.NumberOfBadComputeActors > 0) && (i >= computeNodeCount-stackConfig.NumberOfBadComputeActors)

		if isComputeNode {
			// We have multiple process on the same machine, all wanting to listen on a HTTP port
			// and so we will give each compute node a random open port to listen on.
			fport, err := network.GetFreePort()
			if err != nil {
				return nil, errors.Wrap(err, "failed to get free port for local publisher")
			}
			cfg.Publishers.Types.Local.Port = fport
		}

		cfg.Orchestrator.Enabled = isRequesterNode
		cfg.Compute.Enabled = isComputeNode
		cfg, err = cfg.MergeNew(types.Bacalhau{
			DataDir: filepath.Join(stackConfig.BasePath, nodeID),
			Labels: map[string]string{
				"id":   nodeID,
				"name": fmt.Sprintf("node-%d", i),
				"env":  "devstack",
			},
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to merge new config")
		}

		// create node data dir path
		if err = os.MkdirAll(cfg.DataDir, util.OS_USER_RWX); err != nil {
			return nil, errors.Wrap(err, "failed to create node data directory")
		}

		nodeConfig := node.NodeConfig{
			NodeID:             nodeID,
			CleanupManager:     cm,
			BacalhauConfig:     cfg,
			SystemConfig:       stackConfig.SystemConfig,
			DependencyInjector: stackConfig.NodeDependencyInjector,
			FailureInjectionConfig: models.FailureInjectionConfig{
				IsBadActor: isBadComputeActor,
			},
		}

		// allow overriding configs of some nodes
		if i < len(stackConfig.NodeOverrides) {
			originalConfig := nodeConfig
			nodeConfig = stackConfig.NodeOverrides[i]
			err = mergo.Merge(&nodeConfig, originalConfig)
			if err != nil {
				return nil, err
			}
		}

		var n *node.Node
		n, err = node.NewNode(ctx, nodeConfig, NewMetadataStore())
		if err != nil {
			return nil, err
		}

		// start the node
		err = n.Start(ctx)
		if err != nil {
			return nil, err
		}

		nodes = append(nodes, n)
	}

	// only start profiling after we've set everything up!
	profiler := startProfiling(ctx, stackConfig.CPUProfilingFile, stackConfig.MemoryProfilingFile)
	if profiler != nil {
		cm.RegisterCallbackWithContext(profiler.Close)
	}

	return &DevStack{
		Nodes: nodes,
	}, nil
}

//nolint:funlen
func (stack *DevStack) PrintNodeInfo(ctx context.Context, cm *system.CleanupManager) (string, error) {
	if !config_legacy.DevstackGetShouldPrintInfo() {
		return "", nil
	}

	logString := ""
	devStackAPIPort := fmt.Sprintf("%d", stack.Nodes[0].APIServer.Port)
	devStackAPIHost := stack.Nodes[0].APIServer.Address

	requesterOnlyNodes := 0
	computeOnlyNodes := 0
	hybridNodes := 0
	for nodeIndex, node := range stack.Nodes {
		logString += fmt.Sprintf(`
export BACALHAU_API_HOST_%d=%s
export BACALHAU_API_PORT_%d=%d`,
			nodeIndex,
			stack.Nodes[nodeIndex].APIServer.Address,
			nodeIndex,
			stack.Nodes[nodeIndex].APIServer.Port,
		)

		requesterOnlyNodes += boolToInt(node.IsRequesterNode() && !node.IsComputeNode())
		computeOnlyNodes += boolToInt(node.IsComputeNode() && !node.IsRequesterNode())
		hybridNodes += boolToInt(node.IsRequesterNode() && node.IsComputeNode())
	}

	summaryBuilder := strings.Builder{}
	summaryBuilder.WriteString(fmt.Sprintf(
		"export %s=%s\n",
		config.KeyAsEnvVar(types.APIHostKey),
		devStackAPIHost,
	))
	summaryBuilder.WriteString(fmt.Sprintf(
		"export %s=%s\n",
		config.KeyAsEnvVar(types.APIPortKey),
		devStackAPIPort,
	))

	log.Ctx(ctx).Debug().Msg(logString)

	returnString := fmt.Sprintf(`
Devstack is ready!
No. of requester only nodes: %d
No. of compute only nodes: %d
No. of hybrid nodes: %d
To use the devstack, run the following commands in your shell:

%s

`,
		requesterOnlyNodes,
		computeOnlyNodes,
		hybridNodes,
		summaryBuilder.String())
	return returnString, nil
}

func (stack *DevStack) GetNode(_ context.Context, nodeID string) (
	*node.Node, error) {
	for _, node := range stack.Nodes {
		if node.ID == nodeID {
			return node, nil
		}
	}

	return nil, fmt.Errorf("node not found: %s", nodeID)
}

func (stack *DevStack) GetNodeIds() []string {
	var ids []string
	for _, node := range stack.Nodes {
		ids = append(ids, node.ID)
	}
	return ids
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
