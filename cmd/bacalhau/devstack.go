package bacalhau

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/filecoin-project/bacalhau/pkg/requesternode"

	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
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

	// Set Defaults (probably a better way to do this)
	ODs = newDevStackOptions()

	IsNoop = false

	// For the -f flag
)

func newDevStackOptions() *devstack.DevStackOptions {
	return &devstack.DevStackOptions{
		NumberOfNodes:     3,
		NumberOfBadActors: 0,
		Peer:              "",
		PublicIPFSMode:    false,
		EstuaryAPIKey:     os.Getenv("ESTUARY_API_KEY"),
		LocalNetworkLotus: false,
		SimulatorURL:      "",
	}
}

func init() { //nolint:gochecknoinits // Using init in cobra command is idomatic
	devstackCmd.PersistentFlags().IntVar(
		&ODs.NumberOfNodes, "nodes", ODs.NumberOfNodes,
		`How many nodes should be started in the cluster`,
	)
	devstackCmd.PersistentFlags().IntVar(
		&ODs.NumberOfBadActors, "bad-actors", ODs.NumberOfBadActors,
		`How many nodes should be bad actors`,
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
		&ODs.SimulatorURL, "simulator-url", ODs.SimulatorURL,
		`Use the simulator transport at the given URL`,
	)

	setupJobSelectionCLIFlags(devstackCmd)
	setupCapacityManagerCLIFlags(devstackCmd)
}

var devstackCmd = &cobra.Command{
	Use:     "devstack",
	Short:   "Start a cluster of bacalhau nodes for testing and development",
	Long:    devStackLong,
	Example: devstackExample,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cm := system.NewCleanupManager()
		defer cm.Cleanup()
		ctx := cmd.Context()

		ctx, rootSpan := system.NewRootSpan(ctx, system.GetTracer(), "cmd/bacalhau/devstack")
		defer rootSpan.End()

		cm.RegisterCallback(system.CleanupTraceProvider)

		config.DevstackSetShouldPrintInfo()

		if ODs.NumberOfBadActors >= ODs.NumberOfNodes {
			Fatal(fmt.Sprintf("You cannot have more bad actors (%d) than there are nodes (%d).",
				ODs.NumberOfBadActors, ODs.NumberOfNodes), 1)
		}

		// Context ensures main goroutine waits until killed with ctrl+c:
		ctx, cancel := system.WithSignalShutdown(ctx)
		defer cancel()

		portFileName := filepath.Join(os.TempDir(), "bacalhau-devstack.port")
		pidFileName := filepath.Join(os.TempDir(), "bacalhau-devstack.pid")

		if _, ignore := os.LookupEnv("IGNORE_PID_AND_PORT_FILES"); !ignore {
			_, err := os.Stat(portFileName)
			if err == nil {
				Fatal(fmt.Sprintf("Found file %s - Devstack likely already running", portFileName), 1)
			}
			_, err = os.Stat(pidFileName)
			if err == nil {
				Fatal(fmt.Sprintf("Found file %s - Devstack likely already running", pidFileName), 1)
			}
		}

		computeConfig := getComputeConfig()
		if ODs.LocalNetworkLotus {
			cmd.Println("Note that starting up the Lotus node can take many minutes!")
		}

		var stack *devstack.DevStack
		var stackErr error
		if IsNoop {
			stack, stackErr = devstack.NewNoopDevStack(ctx, cm, *ODs, computeConfig, requesternode.NewDefaultRequesterNodeConfig())
		} else {
			stack, stackErr = devstack.NewStandardDevStack(ctx, cm, *ODs, computeConfig, requesternode.NewDefaultRequesterNodeConfig())
		}
		if stackErr != nil {
			return stackErr
		}

		nodeInfoOutput, err := stack.PrintNodeInfo(ctx)
		if err != nil {
			Fatal(fmt.Sprintf("Failed to print node info: %s", err.Error()), 1)
		}
		cmd.Println(nodeInfoOutput)

		f, err := os.Create(portFileName)
		if err != nil {
			Fatal(fmt.Sprintf("Error writing out port file to %v", portFileName), 1)
		}
		defer os.Remove(portFileName)
		firstNode := stack.Nodes[0]
		_, err = f.WriteString(strconv.Itoa(firstNode.APIServer.Port))
		if err != nil {
			Fatal(fmt.Sprintf("Error writing out port file: %v", portFileName), 1)
		}

		fPid, err := os.Create(pidFileName)
		if err != nil {
			Fatal(fmt.Sprintf("Error writing out pid file to %v", pidFileName), 1)
		}
		defer os.Remove(pidFileName)

		_, err = fPid.WriteString(strconv.Itoa(os.Getpid()))
		if err != nil {
			Fatal(fmt.Sprintf("Error writing out pid file: %v", pidFileName), 1)
		}

		<-ctx.Done() // block until killed

		cmd.Println("Shutting down devstack")
		return nil
	},
}
