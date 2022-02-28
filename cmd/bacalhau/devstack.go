package bacalhau

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/internal"
	"github.com/spf13/cobra"
)

func init() {

}

var devstackCmd = &cobra.Command{
	Use:   "devstack",
	Short: "Start a cluster of 3 bacalhau nodes for testing and development",
	RunE: func(cmd *cobra.Command, args []string) error {

		ctx := context.Background()
		ctxWithCancel, cancelFunction := context.WithCancel(ctx)

		stack, err := internal.NewDevStack(ctxWithCancel, 3)

		if err != nil {
			cancelFunction()
			return err
		}

		for i, node := range stack.Nodes {
			fmt.Printf("\nnode %d:\n", i)
			fmt.Printf("IPFS_PATH=%s ipfs\n", node.IpfsRepo)
			fmt.Printf("go run . --jsonrpc-port=%d list\n", node.JsonRpcPort)
		}

		// wait forever because everything else is running in a goroutine
		select {}

		return nil
	},
}
