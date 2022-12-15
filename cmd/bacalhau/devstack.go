package bacalhau

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/templates"
	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/spf13/cobra"
)

var (
	devStackLong = templates.LongDesc(i18n.T(`
		Start a cluster of nodes and run a job on them.
`))

	//nolint:lll // Documentation
	devstackExample = templates.Examples(i18n.T(`
		# Create a devstack cluster.
		bacalhau devstack
`))
)

func newDevStackOptions() *devstack.DevStackOptions {
	return &devstack.DevStackOptions{
		NumberOfNodes:            3,
		NumberOfBadComputeActors: 0,
		Peer:                     "",
		PublicIPFSMode:           false,
		EstuaryAPIKey:            os.Getenv("ESTUARY_API_KEY"),
		LocalNetworkLotus:        false,
		SimulatorAddr:            "",
		SimulatorMode:            false,
	}
}

func newDevStackCmd() *cobra.Command {
	ODs := newDevStackOptions()
	OS := NewServeOptions()
	IsNoop := false

	devstackCmd := &cobra.Command{
		Use:     "devstack",
		Short:   "Start a cluster of bacalhau nodes for testing and development",
		Long:    devStackLong,
		Example: devstackExample,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDevstack(cmd, ODs, OS, IsNoop)
		},
	}

	devstackCmd.PersistentFlags().IntVar(
		&ODs.NumberOfNodes, "nodes", ODs.NumberOfNodes,
		`How many nodes should be started in the cluster`,
	)
	devstackCmd.PersistentFlags().IntVar(
		&ODs.NumberOfBadComputeActors, "bad-compute-actors", ODs.NumberOfBadComputeActors,
		`How many compute nodes should be bad actors`,
	)
	devstackCmd.PersistentFlags().IntVar(
		&ODs.NumberOfBadComputeActors, "bad-requester-actors", ODs.NumberOfBadRequesterActors,
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
	devstackCmd.PersistentFlags().BoolVar(
		&ODs.LocalNetworkLotus, "lotus-node", ODs.LocalNetworkLotus,
		"Also start a Lotus FileCoin instance",
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

	setupJobSelectionCLIFlags(devstackCmd, OS)
	setupCapacityManagerCLIFlags(devstackCmd, OS)

	return devstackCmd
}

func runDevstack(cmd *cobra.Command, ODs *devstack.DevStackOptions, OS *ServeOptions, IsNoop bool) error {
	cm := system.NewCleanupManager()
	defer cm.Cleanup()
	ctx := cmd.Context()

	ctx, rootSpan := system.NewRootSpan(ctx, system.GetTracer(), "cmd/bacalhau/devstack")
	defer rootSpan.End()

	cm.RegisterCallback(system.CleanupTraceProvider)

	config.DevstackSetShouldPrintInfo()

	if ODs.NumberOfBadComputeActors >= ODs.NumberOfNodes {
		Fatal(cmd, fmt.Sprintf("You cannot have more bad compute actors (%d) than there are nodes (%d).",
			ODs.NumberOfBadComputeActors, ODs.NumberOfNodes), 1)
	}

	// Context ensures main goroutine waits until killed with ctrl+c:
	ctx, cancel := system.WithSignalShutdown(ctx)
	defer cancel()

	portFileName := filepath.Join(os.TempDir(), "bacalhau-devstack.port")
	pidFileName := filepath.Join(os.TempDir(), "bacalhau-devstack.pid")

	if _, ignore := os.LookupEnv("IGNORE_PID_AND_PORT_FILES"); !ignore {
		_, err := os.Stat(portFileName)
		if err == nil {
			Fatal(cmd, fmt.Sprintf("Found file %s - Devstack likely already running", portFileName), 1)
		}
		_, err = os.Stat(pidFileName)
		if err == nil {
			Fatal(cmd, fmt.Sprintf("Found file %s - Devstack likely already running", pidFileName), 1)
		}
	}

	computeConfig := getComputeConfig(OS)
	if ODs.LocalNetworkLotus {
		cmd.Println("Note that starting up the Lotus node can take many minutes!")
	}

	var stack *devstack.DevStack
	var stackErr error
	if IsNoop {
		stack, stackErr = devstack.NewNoopDevStack(ctx, cm, *ODs, computeConfig, node.NewRequesterConfigWithDefaults())
	} else {
		stack, stackErr = devstack.NewStandardDevStack(ctx, cm, *ODs, computeConfig, node.NewRequesterConfigWithDefaults())
	}
	if stackErr != nil {
		return stackErr
	}

	nodeInfoOutput, err := stack.PrintNodeInfo(ctx)
	if err != nil {
		Fatal(cmd, fmt.Sprintf("Failed to print node info: %s", err.Error()), 1)
	}
	cmd.Println(nodeInfoOutput)

	f, err := os.Create(portFileName)
	if err != nil {
		Fatal(cmd, fmt.Sprintf("Error writing out port file to %v", portFileName), 1)
	}
	defer os.Remove(portFileName)
	firstNode := stack.Nodes[0]
	_, err = f.WriteString(strconv.Itoa(firstNode.APIServer.Port))
	if err != nil {
		Fatal(cmd, fmt.Sprintf("Error writing out port file: %v", portFileName), 1)
	}

	fPid, err := os.Create(pidFileName)
	if err != nil {
		Fatal(cmd, fmt.Sprintf("Error writing out pid file to %v", pidFileName), 1)
	}
	defer os.Remove(pidFileName)

	_, err = fPid.WriteString(strconv.Itoa(os.Getpid()))
	if err != nil {
		Fatal(cmd, fmt.Sprintf("Error writing out pid file: %v", pidFileName), 1)
	}

	<-ctx.Done() // block until killed

	cmd.Println("Shutting down devstack")
	return nil
}
