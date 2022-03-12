package bacalhau

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/internal"
	"github.com/filecoin-project/bacalhau/internal/ipfs"
	"github.com/filecoin-project/bacalhau/internal/logger"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var devStackBadActors int

func init() {
	devstackCmd.PersistentFlags().IntVar(
		&devStackBadActors, "bad-actors", 0,
		`How many nodes should be bad actors`,
	)
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

		stack, err := internal.NewDevStack(ctxWithCancel, 3, devStackBadActors)

		if err != nil {
			cancelFunction()
			return err
		}

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			for range c {
				cancelFunction()
				// need some time to let ipfs processes shut down
				time.Sleep(time.Second * 1)
				os.Exit(1)
			}
		}()

		for nodeNumber, node := range stack.Nodes {
			logger.Infof(`
Node %d:
	IPFS_PATH=%s
	JSON_PORT=%d
	bin/bacalhau --jsonrpc-port=%d list
`, nodeNumber, node.IpfsRepo, node.JsonRpcPort, node.JsonRpcPort)
		}

		logger.Infof(`
To add a file, type the following:
file_path="your_file_path_here"
cid=$( IPFS_PATH=%s ipfs add -q $file_path )
`, stack.Nodes[0].IpfsRepo)

		// wait forever because everything else is running in a goroutine
		select {}
	},
}
