package bacalhau

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/filecoin-project/bacalhau/pkg/controller"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	executor_util "github.com/filecoin-project/bacalhau/pkg/executor/util"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	verifier_util "github.com/filecoin-project/bacalhau/pkg/verifier/util"
	"github.com/rs/zerolog/log"

	"github.com/spf13/cobra"
)

var devStackNodes int
var devStackBadActors int
var devStackNoop bool
var devStackPeer string

func init() { // nolint:gochecknoinits // Using init in cobra command is idomatic
	devstackCmd.PersistentFlags().IntVar(
		&devStackNodes, "nodes", 3,
		`How many nodes should be started in the cluster`,
	)
	devstackCmd.PersistentFlags().IntVar(
		&devStackBadActors, "bad-actors", 0,
		`How many nodes should be bad actors`,
	)
	devstackCmd.PersistentFlags().BoolVar(
		&devStackNoop, "noop", false,
		`Use the noop executor and verifier for all jobs`,
	)
	devstackCmd.PersistentFlags().StringVar(
		&devStackPeer, "peer", "",
		`Connect node 0 to another network node`,
	)
}

var devstackCmd = &cobra.Command{
	Use:   "devstack",
	Short: "Start a cluster of bacalhau nodes for testing and development",
	RunE: func(cmd *cobra.Command, args []string) error { // nolintunparam // incorrect lint that is not used

		config.DevstackSetShouldPrintInfo()

		if devStackBadActors > devStackNodes {
			return fmt.Errorf("cannot have more bad actors than there are nodes")
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

			if devStackNoop {
				return executor_util.NewNoopStorageProviders(cm)
			}

			return executor_util.NewStandardStorageProviders(cm, ipfsMultiAddress)
		}

		getExecutors := func(ipfsMultiAddress string, nodeIndex int, ctrl *controller.Controller) (
			map[executor.EngineType]executor.Executor, error) {

			if devStackNoop {
				return executor_util.NewNoopExecutors(cm, noop_executor.ExecutorConfig{})
			}

			return executor_util.NewStandardExecutors(cm,
				ipfsMultiAddress, fmt.Sprintf("devstacknode%d", nodeIndex))
		}

		getVerifiers := func(ipfsMultiAddress string, nodeIndex int, ctrl *controller.Controller) ( //nolint:unparam // nodeIndex will be used in the future
			map[verifier.VerifierType]verifier.Verifier, error) {

			if devStackNoop {
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

		stack, err := devstack.NewDevStack(
			cm,
			devStackNodes,
			devStackBadActors,
			getStorageProviders,
			getExecutors,
			getVerifiers,
			computenode.NewDefaultComputeNodeConfig(),
			devStackPeer,
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
