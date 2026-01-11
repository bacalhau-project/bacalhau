package devstack

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/webui"

	"github.com/bacalhau-project/bacalhau/cmd/util/templates"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"

	"github.com/spf13/cobra"
)

var (
	devStackLong = templates.LongDesc(`
		Start a cluster of nodes and run a job on them.
`)

	devstackExample = templates.Examples(`
		# Create a devstack cluster with a single requester node and 3 compute nodes (Default values)
		bacalhau devstack

		# Create a devstack cluster with a two requester nodes and 10 compute nodes
		bacalhau devstack  --requester-nodes 2 --compute-nodes 10

		# Create a devstack cluster with a single hybrid (requester and compute) nodes
		bacalhau devstack  --requester-nodes 0 --compute-nodes 0 --hybrid-nodes 1

		# Run a devstack and create (or use) the config repo in a specific folder
		bacalhau devstack  --stack-repo ./my-devstack-configuration
`)
)

type options struct {
	// Node counts
	ComputeNodes      int // Number of compute nodes to run
	OrchestratorNodes int // Number of orchestrator nodes to run
	HybridNodes       int // Number of hybrid nodes to run
	BadComputeNodes   int // Number of bad compute nodes

	// Other options
	CPUProfilingFile    string
	MemoryProfilingFile string
	BasePath            string
	RandomPorts         bool // Use random ports for the nodes. Useful to avoid conflicts with active orchestrators
}

func (o *options) devstackOptions() []devstack.ConfigOption {
	opts := []devstack.ConfigOption{
		devstack.WithNumberOfHybridNodes(o.HybridNodes),
		devstack.WithNumberOfRequesterOnlyNodes(o.OrchestratorNodes),
		devstack.WithNumberOfComputeOnlyNodes(o.ComputeNodes),
		devstack.WithNumberOfBadComputeActors(o.BadComputeNodes),
		devstack.WithCPUProfilingFile(o.CPUProfilingFile),
		devstack.WithMemoryProfilingFile(o.MemoryProfilingFile),
		devstack.WithBasePath(o.BasePath),
		devstack.WithUseStandardPorts(!o.RandomPorts),
	}
	return opts
}

func newOptions() *options {
	return &options{
		OrchestratorNodes:   1,
		ComputeNodes:        3,
		CPUProfilingFile:    "",
		MemoryProfilingFile: "",
		BasePath:            "",
	}
}

//nolint:funlen
func NewCmd() *cobra.Command {
	ODs := newOptions()

	devstackCmd := &cobra.Command{
		Use:     "devstack",
		Short:   "Start a cluster of bacalhau nodes for testing and development",
		Long:    devStackLong,
		Example: devstackExample,
		RunE: func(cmd *cobra.Command, _ []string) error {
			defaultConfig, err := config.NewTestConfig()
			if err != nil {
				return fmt.Errorf("failed to create default config: %w", err)
			}
			cfg, err := util.SetupConfig(cmd, config.WithDefault(defaultConfig))
			if err != nil {
				return fmt.Errorf("failed to setup config: %w", err)
			}

			if err = logger.ParseAndConfigureLogging(cfg.Logging.Mode, cfg.Logging.Level); err != nil {
				return fmt.Errorf("failed to configure logging: %w", err)
			}

			// TODO this should be a part of the config.
			telemetry.SetupFromEnvs()

			return runDevstack(cmd, ODs, cfg)
		},
	}

	devstackCmd.PersistentFlags().IntVar(
		&ODs.ComputeNodes, "computes", ODs.ComputeNodes,
		`Number of compute-only nodes to run`,
	)
	devstackCmd.PersistentFlags().IntVar(
		&ODs.OrchestratorNodes, "orchestrators", ODs.OrchestratorNodes,
		`Number of orchestrator-only nodes to run`,
	)
	devstackCmd.PersistentFlags().IntVar(
		&ODs.HybridNodes, "hybrids", ODs.HybridNodes,
		`Number of hybrid nodes (both compute and orchestrator) to run`,
	)
	devstackCmd.PersistentFlags().IntVar(
		&ODs.BadComputeNodes, "bad-computes", ODs.BadComputeNodes,
		`Number of compute nodes that should be bad actors`,
	)

	// Old style flags - hidden from help but still functional
	oldFlags := devstackCmd.PersistentFlags()
	oldFlags.IntVar(
		&ODs.ComputeNodes, "compute-nodes", ODs.ComputeNodes,
		`Number of compute-only nodes to run`,
	)
	_ = oldFlags.MarkHidden("compute-nodes")

	oldFlags.IntVar(
		&ODs.OrchestratorNodes, "requester-nodes", ODs.OrchestratorNodes,
		`Number of orchestrator-only nodes to run`,
	)
	_ = oldFlags.MarkHidden("requester-nodes")

	oldFlags.IntVar(
		&ODs.HybridNodes, "hybrid-nodes", ODs.HybridNodes,
		`Number of hybrid nodes (both compute and orchestrator) to run`,
	)
	_ = oldFlags.MarkHidden("hybrid-nodes")

	oldFlags.IntVar(
		&ODs.BadComputeNodes, "bad-compute-actors", ODs.BadComputeNodes,
		`Number of compute nodes that should be bad actors`,
	)
	_ = oldFlags.MarkHidden("bad-compute-actors")

	devstackCmd.PersistentFlags().StringVar(
		&ODs.CPUProfilingFile, "cpu-profiling-file", ODs.CPUProfilingFile,
		"File to save CPU profiling to",
	)
	devstackCmd.PersistentFlags().StringVar(
		&ODs.MemoryProfilingFile, "memory-profiling-file", ODs.MemoryProfilingFile,
		"File to save memory profiling to",
	)
	devstackCmd.PersistentFlags().StringVar(
		&ODs.BasePath, "stack-repo", ODs.BasePath,
		"Folder to act as the devstack configuration repo",
	)

	devstackCmd.PersistentFlags().BoolVar(&ODs.RandomPorts, "random-ports", ODs.RandomPorts,
		"Use random ports for the nodes. Otherwise will start with standard ports (e.g. 1234). "+
			"Useful to avoid conflicts with active orchestrators",
	)

	return devstackCmd
}

//nolint:funlen
func runDevstack(cmd *cobra.Command, ODs *options, cfg types.Bacalhau) error {
	ctx := cmd.Context()

	cm := util.GetCleanupManager(ctx)
	cm.RegisterCallback(telemetry.Cleanup)

	// Create devstack options and merge with base config
	opts := ODs.devstackOptions()
	opts = append(opts, devstack.WithBacalhauConfigOverride(cfg))

	portFileName := filepath.Join(os.TempDir(), "bacalhau-devstack.port")
	pidFileName := filepath.Join(os.TempDir(), "bacalhau-devstack.pid")

	if _, ignore := os.LookupEnv("IGNORE_PID_AND_PORT_FILES"); !ignore {
		_, err := os.Stat(portFileName)
		if err == nil {
			return fmt.Errorf("found file %s - Devstack likely already running", portFileName)
		}
		_, err = os.Stat(pidFileName)
		if err == nil {
			return fmt.Errorf("found file %s - Devstack likely already running", pidFileName)
		}
	}

	// ensure we either use a temp repo for the devstack, or the repo path provided
	// by the specific devstack flag. Never use the default bacalhau repo.
	baseRepoPath := ODs.BasePath
	if baseRepoPath == "" {
		// We need to clean up the repo when the node shuts down, but we can ONLY
		// do this because we know it is a temporary directory. Do not delete the
		// configured repo if `--stack-repo` was specified
		baseRepoPath, _ = os.MkdirTemp("", "")
		defer func() { _ = os.RemoveAll(baseRepoPath) }()
	}

	stack, err := devstack.Setup(ctx, cm, opts...)
	if err != nil {
		return err
	}

	// start WebUI for the first successful requester node
	for _, n := range stack.Nodes {
		// TODO: move webui creation to node pkg
		if n.IsRequesterNode() {
			webuiConfig := webui.Config{
				APIEndpoint: n.APIServer.GetURI().String(),
				Listen:      cfg.WebUI.Listen,
			}
			webuiServer, err := webui.NewServer(webuiConfig)
			if err != nil {
				log.Warn().Err(err).Msg("Failed to start ui server for this node, trying next")
				continue
			}

			go func() {
				if err := webuiServer.ListenAndServe(ctx); err != nil {
					log.Error().Err(err).Msg("ui server error")
				}
			}()

			break
		}
	}

	cmd.Println(stack.GetStackInfo(ctx))

	f, err := os.Create(portFileName) //nolint:gosec // G304: portFileName from devstack config, application controlled
	if err != nil {
		return fmt.Errorf("error writing out port file to %v: %w", portFileName, err)
	}
	defer func() { _ = os.Remove(portFileName) }()
	firstNode := stack.Nodes[0]
	_, err = f.WriteString(strconv.FormatUint(uint64(firstNode.APIServer.Port), 10))
	if err != nil {
		return fmt.Errorf("error writing out port file: %v: %w", portFileName, err)
	}

	fPid, err := os.Create(pidFileName) //nolint:gosec // G304: pidFileName from devstack config, application controlled
	if err != nil {
		return fmt.Errorf("error writing out pid file to %v: %w", pidFileName, err)
	}
	defer func() { _ = os.Remove(pidFileName) }()

	_, err = fPid.WriteString(strconv.Itoa(os.Getpid()))
	if err != nil {
		return fmt.Errorf("error writing out pid file: %v: %w", pidFileName, err)
	}

	<-ctx.Done() // block until killed

	cmd.Println("\nShutting down devstack")
	return nil
}
