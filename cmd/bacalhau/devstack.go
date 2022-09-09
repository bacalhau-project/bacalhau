package bacalhau

import (
	"fmt"
	"os"
	"strconv"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/templates"
	"github.com/rs/zerolog/log"
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

	setupJobSelectionCLIFlags(devstackCmd)
	setupCapacityManagerCLIFlags(devstackCmd)
}

var devstackCmd = &cobra.Command{
	Use:     "devstack",
	Short:   "Start a cluster of bacalhau nodes for testing and development",
	Long:    devStackLong,
	Example: devstackExample,
	RunE: func(cmd *cobra.Command, args []string) error { // nolintunparam // incorrect lint that is not used
		cm := system.NewCleanupManager()
		defer cm.Cleanup()
		ctx := cmd.Context()

		ctx, rootSpan := system.NewRootSpan(ctx, system.GetTracer(), "cmd/bacalhau/devstack")
		defer rootSpan.End()

		cm.RegisterCallback(system.CleanupTraceProvider)

		config.DevstackSetShouldPrintInfo()

		if ODs.NumberOfBadActors >= ODs.NumberOfNodes {
			return fmt.Errorf("cannot have more bad actors than there are nodes")
		}

		// Context ensures main goroutine waits until killed with ctrl+c:
		ctx, cancel := system.WithSignalShutdown(ctx)
		defer cancel()

		portFileName := "/tmp/bacalhau-devstack.port"
		pidFileName := "/tmp/bacalhau-devstack.pid"

		if _, ignore := os.LookupEnv("IGNORE_PID_AND_PORT_FILES"); !ignore {
			_, err := os.Stat(portFileName)
			if err == nil {
				log.Fatal().Msgf("Found file %s - Devstack likely already running", portFileName)
			}
			_, err = os.Stat(pidFileName)
			if err == nil {
				log.Fatal().Msgf("Found file %s - Devstack likely already running", pidFileName)
			}
		}

		computeNodeConfig := computenode.ComputeNodeConfig{
			JobSelectionPolicy:    getJobSelectionConfig(),
			CapacityManagerConfig: getCapacityManagerConfig(),
		}

		var stack *devstack.DevStack
		var stackErr error
		if IsNoop {
			stack, stackErr = devstack.NewNoopDevStack(ctx, cm, *ODs, computeNodeConfig)
		} else {
			stack, stackErr = devstack.NewStandardDevStack(ctx, cm, *ODs, computeNodeConfig)
		}
		if stackErr != nil {
			return stackErr
		}

		stack.PrintNodeInfo()

		f, err := os.Create(portFileName)
		if err != nil {
			log.Fatal().Msgf("Error writing out port file to %v", portFileName)
		}
		defer os.Remove(portFileName)
		firstNode := stack.Nodes[0]
		_, err = f.WriteString(strconv.Itoa(firstNode.APIServer.Port))
		if err != nil {
			log.Fatal().Msgf("Error writing out port file: %v", portFileName)
		}

		fPid, err := os.Create(pidFileName)
		if err != nil {
			log.Fatal().Msgf("Error writing out pid file to %v", pidFileName)
		}
		defer os.Remove(pidFileName)

		_, err = fPid.WriteString(strconv.Itoa(os.Getpid()))
		if err != nil {
			log.Fatal().Msgf("Error writing out pid file: %v", pidFileName)
		}

		<-ctx.Done() // block until killed

		log.Info().Msg("Shutting down devstack")
		return nil
	},
}
