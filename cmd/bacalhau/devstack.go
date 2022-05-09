package bacalhau

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	ipfs_cli "github.com/filecoin-project/bacalhau/pkg/ipfs/cli"
	"github.com/filecoin-project/bacalhau/pkg/system"

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

		cli := ipfs_cli.NewIPFSCli("")

		result, err := cli.Run([]string{"version"})

		if err != nil {
			log.Error().Msg(fmt.Sprintf("Error running command 'ipfs version': %s", err))
			return err
		}

		if strings.Contains(result, "0.12.0") {
			err = fmt.Errorf("\n********************\nDue to a regression, we do not support 0.12.0. Please install from here:\nhttps://ipfs.io/ipns/dist.ipfs.io/go-ipfs/v0.11.0/go-ipfs_v0.11.0_linux-amd64.tar.gz\n********************\n")
			log.Error().Err(err)
			return err
		}

		ctx, cancelFunction := system.GetCancelContext()

		getExecutors := func(ipfsMultiAddress string, nodeIndex int) (map[string]executor.Executor, error) {
			return devstack.NewDockerIPFSExecutors(ctx, ipfsMultiAddress, fmt.Sprintf("devstacknode%d", nodeIndex))
		}

		stack, err := devstack.NewDevStack(
			ctx,
			devStackNodes,
			devStackBadActors,
			getExecutors,
		)

		if err != nil {
			cancelFunction()
			return err
		}

		stack.PrintNodeInfo()

		// wait forever because everything else is running in a goroutine
		select {}
	},
}
