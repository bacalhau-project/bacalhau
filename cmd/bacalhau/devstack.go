package bacalhau

import (
	"context"
	"fmt"
	"strings"

	"github.com/filecoin-project/bacalhau/internal"
	"github.com/filecoin-project/bacalhau/internal/ipfs"
	"github.com/filecoin-project/bacalhau/internal/logger"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func init() {

}

var devstackCmd = &cobra.Command{
	Use:   "devstack",
	Short: "Start a cluster of 3 bacalhau nodes for testing and development",
	RunE: func(cmd *cobra.Command, args []string) error {

		log, _ := zap.NewProduction()

		result, err := ipfs.IpfsCommand("", []string{"version"})

		if err != nil {
			log.Error(fmt.Sprintf("Error running command 'ipfs version': %s", err))
			return err
		}

		if strings.Contains(result, "0.12.0") {
			err = fmt.Errorf("\n********************\nDue to a regression, we do not support 0.12.0. Please install from here:\nhttps://ipfs.io/ipns/dist.ipfs.io/go-ipfs/v0.11.0/go-ipfs_v0.11.0_linux-amd64.tar.gz\n********************\n")
			log.Error(err.Error())
			return err
		}

		ctx := context.Background()
		ctxWithCancel, cancelFunction := context.WithCancel(ctx)

		stack, err := internal.NewDevStack(ctxWithCancel, 3)

		if err != nil {
			cancelFunction()
			return err
		}

		for nodeNumber, node := range stack.Nodes {
			logger.Infof(`
Node %d:
	IPFS_PATH=%s ipfs
	bin/bacalhau --jsonrpc-port=%d list
`, nodeNumber, node.IpfsRepo, node.JsonRpcPort)
		}

		logger.Infof(`

To add a file, type the following:

file_path="your_file_path_here"
cid=$( IPFS_PATH=%s ipfs add -q $file_path )
`, stack.Nodes[0].IpfsRepo)

		// wait forever because everything else is running in a goroutine
		select {}

		//return nil
	},
}
