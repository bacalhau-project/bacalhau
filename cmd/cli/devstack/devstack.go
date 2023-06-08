package devstack

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/bacalhau-project/bacalhau/cmd/cli/serve"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags"
	"github.com/bacalhau-project/bacalhau/cmd/util/handler"
	computenodeapi "github.com/bacalhau-project/bacalhau/pkg/compute/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/config"
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
		EstuaryAPIKey:              os.Getenv("ESTUARY_API_KEY"),
		SimulatorAddr:              "",
		SimulatorMode:              false,
		CPUProfilingFile:           "",
		MemoryProfilingFile:        "",
	}
}

func NewCmd() *cobra.Command {
	ODs := newDevStackOptions()
	OS := serve.NewServeOptions()

	// make sure serve options point to local mode
	OS.PeerConnect = serve.DefaultPeerConnect
	OS.PrivateInternalIPFS = true

	IsNoop := false

	devstackCmd := &cobra.Command{
		Use:     "devstack",
		Short:   "Start a cluster of bacalhau nodes for testing and development",
		Long:    devStackLong,
		Example: devstackExample,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err, exitcode := runDevstack(cmd, ODs, OS, IsNoop); err != nil {
				handler.Fatal(cmd, err, exitcode)
			}
			return nil
		},
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
		&IsNoop, "noop", false,
		`Use the noop executor and verifier for all jobs`,
	)
	devstackCmd.PersistentFlags().StringVar(
		&ODs.Peer, "peer", ODs.Peer,
		`Connect node 0 to another network node`,
	)
	devstackCmd.PersistentFlags().StringVar(
		&ODs.SimulatorAddr, "simulator-addr", ODs.SimulatorAddr,
		`Use the simulator transport at the given node multi addr`,
	)
	devstackCmd.PersistentFlags().BoolVar(
		&ODs.SimulatorMode, "simulator-mode", false,
		`If set, one of the nodes will act as a simulator and will proxy all requests to the other nodes`,
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
		"Local paths that are allowed to be mounted into jobs",
	)
	devstackCmd.PersistentFlags().Var(
		flags.URLFlag(&OS.ExternalVerifierHook, "http"), "external-verifier-http",
		"An HTTP URL to which the verification request should be posted for jobs using the 'external' verifier. "+
			"The 'external' verifier will not be enabled if this is unset.",
	)

	devstackCmd.Flags().AddFlagSet(flags.JobSelectionCLIFlags(&OS.JobSelectionPolicy))
	devstackCmd.Flags().AddFlagSet(flags.DisabledFeatureCLIFlags(&ODs.DisabledFeatures))
	serve.SetupCapacityManagerCLIFlags(devstackCmd, OS)

	return devstackCmd
}

//nolint:gocyclo
func runDevstack(cmd *cobra.Command, ODs *devstack.DevStackOptions, OS *serve.ServeOptions, IsNoop bool) (error, int) {
	ctx := cmd.Context()

	cm := handler.GetCleanupManager(ctx)

	// make sure we don't run devstack with a custom IPFS path - that must be used only with serve
	if os.Getenv("BACALHAU_SERVE_IPFS_PATH") != "" {
		return fmt.Errorf("unset BACALHAU_SERVE_IPFS_PATH in your environment to run devstack"), handler.ExitError
	}

	cm.RegisterCallback(telemetry.Cleanup)

	config.DevstackSetShouldPrintInfo()

	totalComputeNodes := ODs.NumberOfComputeOnlyNodes + ODs.NumberOfHybridNodes
	totalRequesterNodes := ODs.NumberOfRequesterOnlyNodes + ODs.NumberOfHybridNodes
	if ODs.NumberOfBadComputeActors > totalComputeNodes {
		return fmt.Errorf("you cannot have more bad compute actors (%d) than there are nodes (%d)", ODs.NumberOfBadComputeActors, totalComputeNodes), handler.ExitError
	}
	if ODs.NumberOfBadRequesterActors > totalRequesterNodes {
		return fmt.Errorf("you cannot have more bad requester actors (%d) than there are nodes (%d)", ODs.NumberOfBadRequesterActors, totalRequesterNodes), handler.ExitError
	}

	portFileName := filepath.Join(os.TempDir(), "bacalhau-devstack.port")
	pidFileName := filepath.Join(os.TempDir(), "bacalhau-devstack.pid")

	if _, ignore := os.LookupEnv("IGNORE_PID_AND_PORT_FILES"); !ignore {
		_, err := os.Stat(portFileName)
		if err == nil {
			return fmt.Errorf("found file %s - Devstack likely already running", portFileName), handler.ExitError
		}
		_, err = os.Stat(pidFileName)
		if err == nil {
			return fmt.Errorf("found file %s - Devstack likely already running", pidFileName), handler.ExitError
		}
	}

	computeConfig := serve.GetComputeConfig(OS)
	requestorConfig := serve.GetRequesterConfig(OS)

	var stack *devstack.DevStack
	var stackErr error
	if IsNoop {
		stack, stackErr = devstack.NewNoopDevStack(ctx, cm, *ODs, computeConfig, requestorConfig)
	} else {
		stack, stackErr = devstack.NewStandardDevStack(ctx, cm, *ODs, computeConfig, requestorConfig)
	}
	if stackErr != nil {
		return stackErr, handler.ExitError
	}

	nodeInfoOutput, err := stack.PrintNodeInfo(ctx, cm)
	if err != nil {
		return fmt.Errorf("failed to print node info: %w", err), handler.ExitError
	}
	cmd.Println(nodeInfoOutput)

	f, err := os.Create(portFileName)
	if err != nil {
		return fmt.Errorf("error writing out port file to %v: %w", portFileName, err), handler.ExitError
	}
	defer os.Remove(portFileName)
	firstNode := stack.Nodes[0]
	_, err = f.WriteString(strconv.FormatUint(uint64(firstNode.APIServer.Port), 10))
	if err != nil {
		return fmt.Errorf("error writing out port file: %v: %w", portFileName, err), handler.ExitError
	}

	fPid, err := os.Create(pidFileName)
	if err != nil {
		return fmt.Errorf("error writing out pid file to %v: %w", pidFileName, err), handler.ExitError
	}
	defer os.Remove(pidFileName)

	_, err = fPid.WriteString(strconv.Itoa(os.Getpid()))
	if err != nil {
		return fmt.Errorf("error writing out pid file: %v: %w", pidFileName, err), handler.ExitError
	}

	if handler.LoggingMode == logger.LogModeStation {
		for _, node := range stack.Nodes {
			if node.IsComputeNode() {
				cmd.Printf("API: %s\n", node.APIServer.GetURI().JoinPath(computenodeapi.APIPrefix, computenodeapi.APIDebugSuffix))
			}
		}
	}

	<-ctx.Done() // block until killed

	cmd.Println("\nShutting down devstack")
	return nil, handler.ExitSuccess
}
