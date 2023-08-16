package devstack

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/viper"
	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags/configflags"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/repo"

	"github.com/bacalhau-project/bacalhau/cmd/cli/serve"
	"github.com/bacalhau-project/bacalhau/cmd/util"
	computenodeapi "github.com/bacalhau-project/bacalhau/pkg/compute/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"

	"github.com/spf13/cobra"
)

var (
	devStackLong = templates.LongDesc(i18n.T(`
		Start a cluster of nodes and run a job on them.
`))

	//nolint:lll // Documentation
	devstackExample = templates.Examples(i18n.T(`
		# Create a devstack cluster with a single requester node and 3 compute nodes (Default values)
		bacalhau devstack

		# Create a devstack cluster with a two requester nodes and 10 compute nodes
		bacalhau devstack  --requester-nodes 2 --compute-nodes 10

		# Create a devstack cluster with a single hybrid (requester and compute) nodes
		bacalhau devstack  --requester-nodes 0 --compute-nodes 0 --hybrid-nodes 1
`))
)

func newDevStackOptions() *devstack.DevStackOptions {
	return &devstack.DevStackOptions{
		NumberOfRequesterOnlyNodes: 1,
		NumberOfComputeOnlyNodes:   3,
		NumberOfBadComputeActors:   0,
		Peer:                       "",
		PublicIPFSMode:             false,
		CPUProfilingFile:           "",
		MemoryProfilingFile:        "",
		NodeInfoPublisherInterval:  node.TestNodeInfoPublishConfig,
	}
}

func NewCmd() *cobra.Command {
	ODs := newDevStackOptions()
	IsNoop := false
	devstackFlags := map[string][]configflags.Definition{
		"job-selection":    configflags.JobSelectionFlags,
		"disable-features": configflags.DisabledFeatureFlags,
		"capacity":         configflags.CapacityFlags,
	}

	devstackCmd := &cobra.Command{
		Use:     "devstack",
		Short:   "Start a cluster of bacalhau nodes for testing and development",
		Long:    devStackLong,
		Example: devstackExample,
		PreRun: func(cmd *cobra.Command, _ []string) {
			if err := configflags.BindFlags(cmd, devstackFlags); err != nil {
				util.Fatal(cmd, err, 1)
			}
		},
		Run: func(cmd *cobra.Command, _ []string) {
			if err := runDevstack(cmd, ODs, IsNoop); err != nil {
				util.Fatal(cmd, err, 1)
			}
		},
	}

	if err := configflags.RegisterFlags(devstackCmd, devstackFlags); err != nil {
		util.Fatal(devstackCmd, err, 1)
	}

	devstackCmd.PersistentFlags().IntVar(
		&ODs.NumberOfHybridNodes, "hybrid-nodes", ODs.NumberOfHybridNodes,
		`How many hybrid (requester and compute) nodes should be started in the cluster`,
	)
	devstackCmd.PersistentFlags().IntVar(
		&ODs.NumberOfRequesterOnlyNodes, "requester-nodes", ODs.NumberOfRequesterOnlyNodes,
		`How many requester only nodes should be started in the cluster`,
	)
	devstackCmd.PersistentFlags().IntVar(
		&ODs.NumberOfComputeOnlyNodes, "compute-nodes", ODs.NumberOfComputeOnlyNodes,
		`How many compute only nodes should be started in the cluster`,
	)
	devstackCmd.PersistentFlags().IntVar(
		&ODs.NumberOfBadComputeActors, "bad-compute-actors", ODs.NumberOfBadComputeActors,
		`How many compute nodes should be bad actors`,
	)
	devstackCmd.PersistentFlags().IntVar(
		&ODs.NumberOfBadRequesterActors, "bad-requester-actors", ODs.NumberOfBadRequesterActors,
		`How many requester nodes should be bad actors`,
	)
	devstackCmd.PersistentFlags().BoolVar(
		&IsNoop, "Noop", false,
		`Use the noop executor for all jobs`,
	)
	devstackCmd.PersistentFlags().StringVar(
		&ODs.Peer, "peer", ODs.Peer,
		`Connect node 0 to another network node`,
	)
	devstackCmd.PersistentFlags().BoolVar(
		&ODs.PublicIPFSMode, "public-ipfs", ODs.PublicIPFSMode,
		`Connect devstack to public IPFS`,
	)
	devstackCmd.PersistentFlags().StringVar(
		&ODs.CPUProfilingFile, "cpu-profiling-file", ODs.CPUProfilingFile,
		"File to save CPU profiling to",
	)
	devstackCmd.PersistentFlags().StringVar(
		&ODs.MemoryProfilingFile, "memory-profiling-file", ODs.MemoryProfilingFile,
		"File to save memory profiling to",
	)
	devstackCmd.PersistentFlags().StringSliceVar(
		&ODs.AllowListedLocalPaths, "allow-listed-local-paths", ODs.AllowListedLocalPaths,
		"Local paths that are allowed to be mounted into jobs. Multiple paths can be specified by using this flag multiple times.",
	)
	devstackCmd.PersistentFlags().BoolVar(
		&ODs.ExecutorPlugins, "pluggable-executors", ODs.ExecutorPlugins,
		"Will use pluggable executors when set to true",
	)

	return devstackCmd
}

//nolint:gocyclo
func runDevstack(cmd *cobra.Command, ODs *devstack.DevStackOptions, IsNoop bool) error {
	ctx := cmd.Context()

	cm := util.GetCleanupManager(ctx)

	fsRepo, err := repo.NewFS(viper.GetString("repo"))
	if err != nil {
		return err
	}
	if err := fsRepo.Open(); err != nil {
		return err
	}

	// make sure we don't run devstack with a custom IPFS path - that must be used only with serve
	if os.Getenv("BACALHAU_SERVE_IPFS_PATH") != "" {
		return fmt.Errorf("unset BACALHAU_SERVE_IPFS_PATH in your environment to run devstack")
	}

	cm.RegisterCallback(telemetry.Cleanup)

	config.DevstackSetShouldPrintInfo()

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

	computeConfig, err := serve.GetComputeConfig()
	if err != nil {
		return err
	}
	requestorConfig, err := serve.GetRequesterConfig()
	if err != nil {
		return err
	}

	options := append(ODs.Options(),
		devstack.WithComputeConfig(computeConfig),
		devstack.WithRequesterConfig(requestorConfig),
	)
	if IsNoop {
		options = append(options, devstack.WithDependencyInjector(devstack.NewNoopNodeDependencyInjector()))
	} else if ODs.ExecutorPlugins {
		options = append(options, devstack.WithDependencyInjector(node.NewExecutorPluginNodeDependencyInjector()))
	} else {
		options = append(options, devstack.WithDependencyInjector(node.NewStandardNodeDependencyInjector()))
	}
	stack, err := devstack.Setup(ctx, cm, fsRepo, options...)
	if err != nil {
		return err
	}

	nodeInfoOutput, err := stack.PrintNodeInfo(ctx, cm)
	if err != nil {
		return fmt.Errorf("failed to print node info: %w", err)
	}
	cmd.Println(nodeInfoOutput)

	f, err := os.Create(portFileName)
	if err != nil {
		return fmt.Errorf("error writing out port file to %v: %w", portFileName, err)
	}
	defer os.Remove(portFileName)
	firstNode := stack.Nodes[0]
	_, err = f.WriteString(strconv.FormatUint(uint64(firstNode.APIServer.Port), 10))
	if err != nil {
		return fmt.Errorf("error writing out port file: %v: %w", portFileName, err)
	}

	fPid, err := os.Create(pidFileName)
	if err != nil {
		return fmt.Errorf("error writing out pid file to %v: %w", pidFileName, err)
	}
	defer os.Remove(pidFileName)

	_, err = fPid.WriteString(strconv.Itoa(os.Getpid()))
	if err != nil {
		return fmt.Errorf("error writing out pid file: %v: %w", pidFileName, err)
	}

	if config.GetLogMode() == logger.LogModeStation {
		for _, node := range stack.Nodes {
			if node.IsComputeNode() {
				cmd.Printf("API: %s\n", node.APIServer.GetURI().JoinPath(computenodeapi.APIPrefix, computenodeapi.APIDebugSuffix))
			}
		}
	}

	<-ctx.Done() // block until killed

	cmd.Println("\nShutting down devstack")
	return nil
}
