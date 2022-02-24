package bacalhau

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/internal"
	"github.com/filecoin-project/bacalhau/internal/ipfs"
	"github.com/phayes/freeport"
	"github.com/spf13/cobra"
)

func init() {

}

var devstackCmd = &cobra.Command{
	Use:   "devstack",
	Short: "Start a cluster of 3 bacalhau nodes for testing and development",
	RunE: func(cmd *cobra.Command, args []string) error {

		ctx := context.Background()
		nodes := []*internal.ComputeNode{}

		ipfsMultiAddresses := []string{}
		ipfsRepos := []string{}

		// create 3 bacalhau compute nodes
		for i := range []int{0, 1, 2} {
			computePort, err := freeport.GetFreePort()
			if err != nil {
				return err
			}
			node, err := internal.NewComputeNode(ctx, computePort)
			if err != nil {
				return err
			}

			nodes = append(nodes, node)

			jsonRpcPort, err := freeport.GetFreePort()
			if err != nil {
				return err
			}

			go (func() {
				internal.RunBacalhauJsonRpcServer("0.0.0.0", jsonRpcPort, node)
			})()

			connectToMultiAddress := ""

			// if we started any ipfs servers already, use the first one
			if len(ipfsMultiAddresses) > 0 {
				connectToMultiAddress = ipfsMultiAddresses[0]
			}

			ipfsRepo, ipfsMultiaddress, err := ipfs.StartBacalhauDevelopmentIpfsServer(connectToMultiAddress)
			if err != nil {
				return err
			}

			fmt.Printf("ipfs multiaddress: %s\n", ipfsMultiaddress)
			fmt.Printf("ipfs repo: %s\n", ipfsRepo)
			ipfsMultiAddresses = append(ipfsMultiAddresses, ipfsMultiaddress)
			ipfsRepos = append(ipfsRepos, ipfsRepo)

			node.IpfsRepo = ipfsRepo

			if i > 0 {
				// connect to the first node
				connectToAddress := fmt.Sprintf("%s/p2p/%s", nodes[0].Host.Addrs()[0].String(), nodes[0].Host.ID())
				fmt.Printf("bacalhau multiaddress: %s\n", connectToAddress)
				err = node.Connect(connectToAddress)
				if err != nil {
					return err
				}
			}
		}

		// wait forever because everything else is running in a goroutine
		select {}

		return nil
	},
}
