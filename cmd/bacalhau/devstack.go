package bacalhau

import (
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
	RunE: func(cmd *cobra.Command, args []string) error { // nolint

		if devStackBadActors > devStackNodes {
			return fmt.Errorf("Cannot have more bad actors than there are nodes")
		}

		cancelContext := system.GetCancelContextWithSignals()

		getExecutors := func(ipfsMultiAddress string, nodeIndex int) (map[string]executor.Executor, error) {
			return executor.NewDockerIPFSExecutors(cancelContext, ipfsMultiAddress, fmt.Sprintf("devstacknode%d", nodeIndex))
		}

		getVerifiers := func(ipfsMultiAddress string, nodeIndex int) (map[string]verifier.Verifier, error) {
			return verifier.NewIPFSVerifiers(cancelContext, ipfsMultiAddress)
		}

		stack, err := devstack.NewDevStack(
			cancelContext,
			devStackNodes,
			devStackBadActors,
			getExecutors,
			getVerifiers,
		)

		if err != nil {
			cancelContext.Stop()
			return err
		}

		stack.PrintNodeInfo()

		select {}
	},
}
