package bacalhau

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"

	"github.com/filecoin-project/bacalhau/pkg/capacitymanager"
	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/filecoin-project/bacalhau/pkg/controller"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	executor_util "github.com/filecoin-project/bacalhau/pkg/executor/util"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/templates"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	verifier_util "github.com/filecoin-project/bacalhau/pkg/verifier/util"
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
	ODs = NewDevStackOptions()

	// For the -f flag
)

type DevStackOptions struct {
	NumberOfNodes     int    // Number of nodes to start in the cluster
	NumberOfBadActors int    // Number of nodes to be bad actors
	IsNoop            bool   // Noop executor and verifier for all jobs
	Peer              string // Connect node 0 to another network node
}

func NewDevStackOptions() *DevStackOptions {
	return &DevStackOptions{
		NumberOfNodes:     3,
		NumberOfBadActors: 0,
		IsNoop:            false,
		Peer:              "",
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
		&ODs.IsNoop, "noop", false,
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
	RunE: func(cmd *cobra.Command, args []string) error { //nolint:unparam // incorrect lint that is not used
		// devstack always records a cpu profile, it will be generally useful.
		cpuprofile := "/tmp/bacalhau-devstack-cpu.prof"
		f, err := os.Create(cpuprofile)
		if err != nil {
			log.Fatal().Msgf("could not create CPU profile: ", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal().Msgf("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()

		memprofile := "/tmp/bacalhau-devstack-mem.prof"
		f, err = os.Create(memprofile)
		if err != nil {
			log.Fatal().Msgf("could not create memory profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		runtime.GC()    // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal().Msgf("could not write memory profile: ", err)
		}

		config.DevstackSetShouldPrintInfo()

		if ODs.NumberOfBadActors >= ODs.NumberOfNodes {
			return fmt.Errorf("must have more nodes (%d) than bad actors (%d)", ODs.NumberOfNodes, ODs.NumberOfBadActors)
		}

		// Cleanup manager ensures that resources are freed before exiting:
		cm := system.NewCleanupManager()
		cm.RegisterCallback(system.CleanupTracer)
		defer cm.Cleanup()

		// Context ensures main goroutine waits until killed with ctrl+c:
		ctx, cancel := system.WithSignalShutdown(context.Background())
		defer cancel()

		getStorageProviders := func(ipfsMultiAddress string, nodeIndex int) (
			map[storage.StorageSourceType]storage.StorageProvider, error) {

			if ODs.IsNoop {
				return executor_util.NewNoopStorageProviders(cm)
			}

			return executor_util.NewStandardStorageProviders(cm, ipfsMultiAddress)
		}

		getExecutors := func(ipfsMultiAddress string, nodeIndex int, ctrl *controller.Controller) (
			map[executor.EngineType]executor.Executor, error) {

			if ODs.IsNoop {
				return executor_util.NewNoopExecutors(cm, noop_executor.ExecutorConfig{})
			}

			return executor_util.NewStandardExecutors(cm,
				ipfsMultiAddress, fmt.Sprintf("devstacknode%d", nodeIndex))
		}

		// nodeIndex will be used in the future
		getVerifiers := func(ipfsMultiAddress string, nodeIndex int, ctrl *controller.Controller) ( //nolint:unparam,lll
			map[verifier.VerifierType]verifier.Verifier, error) {

			if ODs.IsNoop {
				return verifier_util.NewNoopVerifiers(cm)
			}

			jobLoader := func(ctx context.Context, id string) (executor.Job, error) {
				return ctrl.GetJob(ctx, id)
			}
			stateLoader := func(ctx context.Context, id string) (executor.JobState, error) {
				return ctrl.GetJobState(ctx, id)
			}
			return verifier_util.NewIPFSVerifiers(cm, ipfsMultiAddress, jobLoader, stateLoader)
		}

		jobSelectionPolicy := getJobSelectionConfig()
		totalResourceLimit, jobResourceLimit := getCapacityManagerConfig()

		computeNodeConfig := computenode.ComputeNodeConfig{
			JobSelectionPolicy: jobSelectionPolicy,
			CapacityManagerConfig: capacitymanager.Config{
				ResourceLimitTotal: totalResourceLimit,
				ResourceLimitJob:   jobResourceLimit,
			},
		}

		stack, err := devstack.NewDevStack(
			cm,
			ODs.NumberOfNodes,
			ODs.NumberOfBadActors,
			getStorageProviders,
			getExecutors,
			getVerifiers,
			computeNodeConfig,
			ODs.Peer,
			false,
		)
		if err != nil {
			return err
		}

		stack.PrintNodeInfo()

		portFileName := "/tmp/bacalhau-devstack.port"
		_, err = os.Stat(portFileName)
		if err == nil {
			log.Fatal().Msgf("Found file %s - Devstack likely already running", portFileName)
		}
		f, err = os.Create(portFileName)
		if err != nil {
			log.Fatal().Msgf("Error writing out port file to %v", portFileName)
		}
		defer os.Remove(portFileName)
		firstNode := stack.Nodes[0]
		_, err = f.WriteString(strconv.Itoa(firstNode.APIServer.Port))
		if err != nil {
			log.Fatal().Msgf("Error writing out port file: %v", portFileName)
		}

		pidFileName := "/tmp/bacalhau-devstack.pid"
		_, err = os.Stat(pidFileName)
		if err == nil {
			log.Fatal().Msgf("Found file %s - Devstack likely already running", pidFileName)
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
		return nil
	},
}
