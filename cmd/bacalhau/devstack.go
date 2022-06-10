package bacalhau

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"

	"github.com/spf13/cobra"
)

var devStackNodes int
var devStackBadActors int

func init() {
	devstackCmd.PersistentFlags().IntVar(
		&devStackNodes, "nodes", 3,
		`How many nodes should be started in the cluster`,
	)
	devstackCmd.PersistentFlags().IntVar(
		&devStackBadActors, "bad-actors", 0,
		`How many nodes should be bad actors`,
	)
}

var devstackCmd = &cobra.Command{
	Use:   "devstack",
	Short: "Start a cluster of bacalhau nodes for testing and development",
	RunE: func(cmd *cobra.Command, args []string) error {
		if devStackBadActors > devStackNodes {
			return fmt.Errorf("Cannot have more bad actors than there are nodes")
		}

		// Cleanup manager ensures that resources are freed before exiting:
		cm := system.NewCleanupManager()
		defer cm.Cleanup()

		// Context ensures main goroutine waits until killed with ctrl+c:
		ctx, cancel := system.WithSignalShutdown(context.Background())
		defer cancel()

		getExecutors := func(ipfsMultiAddress string, nodeIndex int) (map[string]executor.Executor, error) {
			return executor.NewDockerIPFSExecutors(cm, ipfsMultiAddress, fmt.Sprintf("devstacknode%d", nodeIndex))
		}

		getVerifiers := func(ipfsMultiAddress string, nodeIndex int) (map[string]verifier.Verifier, error) {
			return verifier.NewIPFSVerifiers(cm, ipfsMultiAddress)
		}

		stack, err := devstack.NewDevStack(
			cm,
			devStackNodes,
			devStackBadActors,
			getExecutors,
			getVerifiers,
		)
		if err != nil {
			return err
		}

		stack.PrintNodeInfo()
		<-ctx.Done() // block until killed
		return nil
	},
}
