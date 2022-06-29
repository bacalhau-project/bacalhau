package bacalhau

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	executor_util "github.com/filecoin-project/bacalhau/pkg/executor/util"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	verifier_util "github.com/filecoin-project/bacalhau/pkg/verifier/util"

	"github.com/spf13/cobra"
)

var devStackNodes int
var devStackBadActors int
var devStackNoopExecutor bool

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
		&devStackNoopExecutor, "noop-executor", false,
		`Use the noop executor for all jobs`,
	)
}

var devstackCmd = &cobra.Command{
	Use:   "devstack",
	Short: "Start a cluster of bacalhau nodes for testing and development",
	RunE: func(cmd *cobra.Command, args []string) error { // nolintunparam // incorrect lint that is not used
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

		getExecutors := func(ipfsMultiAddress string, nodeIndex int) (
			map[executor.EngineType]executor.Executor, error) {

			if devStackNoopExecutor {
				return executor_util.NewNoopExecutors(cm)
			}

			return executor_util.NewDockerIPFSExecutors(cm,
				ipfsMultiAddress, fmt.Sprintf("devstacknode%d", nodeIndex))
		}

		getVerifiers := func(ipfsMultiAddress string, nodeIndex int) (
			map[verifier.VerifierType]verifier.Verifier, error) {

			return verifier_util.NewIPFSVerifiers(cm, ipfsMultiAddress)
		}

		stack, err := devstack.NewDevStack(
			cm,
			devStackNodes,
			devStackBadActors,
			getExecutors,
			getVerifiers,
			computenode.NewDefaultJobSelectionPolicy(),
		)
		if err != nil {
			return err
		}

		stack.PrintNodeInfo()
		<-ctx.Done() // block until killed
		return nil
	},
}
