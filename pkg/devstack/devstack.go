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
	"github.com/bacalhau-project/bacalhau/pkg/lib/network"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type DevStack struct {
	Nodes                 []*node.Node
	config                *DevStackConfig
	nextNodeID            int
	orchestratorEndpoints []string
}

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

	// Create empty devstack to start adding nodes to
	stack := &DevStack{
		Nodes:                 []*node.Node{},
		config:                stackConfig,
		nextNodeID:            0,
		orchestratorEndpoints: []string{},
	}

	totalNodeCount := stackConfig.NumberOfHybridNodes + stackConfig.NumberOfRequesterOnlyNodes + stackConfig.NumberOfComputeOnlyNodes
	requesterNodeCount := stackConfig.NumberOfHybridNodes + stackConfig.NumberOfRequesterOnlyNodes
	computeNodeCount := stackConfig.NumberOfHybridNodes + stackConfig.NumberOfComputeOnlyNodes

	for i := 0; i < totalNodeCount; i++ {
		isRequesterNode := i < requesterNodeCount
		isComputeNode := (totalNodeCount - i) <= computeNodeCount
		isBadComputeActor := (stackConfig.NumberOfBadComputeActors > 0) && (i >= computeNodeCount-stackConfig.NumberOfBadComputeActors)
		var nodeOverride *node.NodeConfig
		if i < len(stackConfig.NodeOverrides) {
			nodeOverride = &stackConfig.NodeOverrides[i]
		}

		_, err := stack.JoinNode(ctx, cm, JoinNodeOptions{
			IsRequester:     isRequesterNode,
			IsCompute:       isComputeNode,
			BadComputeActor: isBadComputeActor,
			ConfigOverride:  nodeOverride,
		})
		if err != nil {
			return nil, err
		}
	}

	// Add hybrid nodes first (they can act as both orchestrator and compute)

	// only start profiling after we've set everything up!
	p := startProfiling(ctx, stackConfig.CPUProfilingFile, stackConfig.MemoryProfilingFile)
	if p != nil {
		cm.RegisterCallbackWithContext(p.Close)
	}

	return stack, nil
}

type JoinNodeOptions struct {
	IsRequester     bool
	IsCompute       bool
	BadComputeActor bool // Make this node a bad compute actor for testing
	ConfigOverride  *node.NodeConfig
}

func (stack *DevStack) JoinComputeNode(ctx context.Context, cm *system.CleanupManager) (*node.Node, error) {
	return stack.JoinNode(ctx, cm, JoinNodeOptions{IsCompute: true})
}

func (stack *DevStack) JoinOrchestratorNode(ctx context.Context, cm *system.CleanupManager) (*node.Node, error) {
	return stack.JoinNode(ctx, cm, JoinNodeOptions{IsRequester: true})
}

func (stack *DevStack) JoinHybridNode(ctx context.Context, cm *system.CleanupManager) (*node.Node, error) {
	return stack.JoinNode(ctx, cm, JoinNodeOptions{IsRequester: true, IsCompute: true})
}

func (stack *DevStack) JoinNode(ctx context.Context, cm *system.CleanupManager, options JoinNodeOptions) (*node.Node, error) {
	nodeID := fmt.Sprintf("node-%d", stack.nextNodeID)
	ctx = logger.ContextWithNodeIDLogger(ctx, nodeID)

	log.Ctx(ctx).Debug().Msgf("Creating joined node %s as {RequesterNode: %t, ComputeNode: %t}",
		nodeID, options.IsRequester, options.IsCompute)

	// Clone the base config from the devstack
	cfg := stack.config.BacalhauConfig
	var err error

	// Get orchestrator endpoints from existing orchestrator nodes for compute nodes
	if options.IsCompute && !options.IsRequester {
		orchestrators := stack.orchestratorEndpoints
		if len(orchestrators) == 0 {
			return nil, fmt.Errorf("cannot add compute node: no orchestrators found in the existing devstack")
		}
		cfg.Compute.Orchestrators = orchestrators
	}

	// Configure ports - always use random ports to avoid conflicts
	if options.IsRequester {
		if stack.config.UseStandardPorts {
			cfg.Orchestrator.Port = cfg.Orchestrator.Port + stack.nextNodeID
		} else {
			if cfg.Orchestrator.Port, err = network.GetFreePort(); err != nil {
				return nil, errors.Wrap(err, "failed to get free port for nats server")
			}
		}
		// Store the new orchestrator endpoint for future compute nodes
		newOrchestratorEndpoint := fmt.Sprintf("127.0.0.1:%d", cfg.Orchestrator.Port)
		stack.orchestratorEndpoints = append(stack.orchestratorEndpoints, newOrchestratorEndpoint)
	}

	if stack.config.UseStandardPorts {
		cfg.API.Port = cfg.API.Port + stack.nextNodeID
		// add one more if using an external orchestrator to avoid port conflict
		if len(stack.orchestratorEndpoints) == 0 {
			cfg.API.Port += 1
		}
	} else {
		if cfg.API.Port, err = network.GetFreePort(); err != nil {
			return nil, errors.Wrap(err, "failed to get free port for API server")
		}
	}

	if options.IsCompute {
		freePort, err := network.GetFreePort()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get free port for local publisher")
		}
		cfg.Publishers.Types.Local.Port = freePort
	}

	// Configure node capabilities
	cfg.Orchestrator.Enabled = options.IsRequester
	cfg.Compute.Enabled = options.IsCompute

	// Set data directory and labels
	cfg, err = cfg.MergeNew(types.Bacalhau{
		DataDir: filepath.Join(stack.config.BasePath, nodeID),
		Labels: map[string]string{
			"id":   nodeID,
			"name": nodeID,
			"env":  "devstack",
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to merge config")
	}

	// Create node data directory
	if err = os.MkdirAll(cfg.DataDir, util.OS_USER_RWX); err != nil {
		return nil, errors.Wrap(err, "failed to create node data directory")
	}

	// Create node config
	nodeConfig := node.NodeConfig{
		NodeID:             nodeID,
		CleanupManager:     cm,
		BacalhauConfig:     cfg,
		SystemConfig:       stack.config.SystemConfig,
		DependencyInjector: stack.config.NodeDependencyInjector,
		FailureInjectionConfig: models.FailureInjectionConfig{
			IsBadActor: options.BadComputeActor,
		},
	}

	// allow overriding configs of some nodes
	if options.ConfigOverride != nil {
		originalConfig := nodeConfig
		nodeConfig = *options.ConfigOverride
		err = mergo.Merge(&nodeConfig, originalConfig)
		if err != nil {
			return nil, err
		}
	}

	// Create and start the node
	n, err := node.NewNode(ctx, nodeConfig, NewMetadataStore())
	if err != nil {
		return nil, fmt.Errorf("failed to create node %s: %w", nodeID, err)
	}

	if err = n.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start node %s: %w", nodeID, err)
	}

	stack.Nodes = append(stack.Nodes, n)
	stack.nextNodeID++
	return n, nil
}

func (stack *DevStack) GetStackInfo(ctx context.Context) string {
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
	return returnString
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
