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

		// create 3 bacalhau compute nodes
		//for i := range []int{0} {
		for range []int{0} {
			computePort, err := freeport.GetFreePort()
			if err != nil {
				return err
			}
			node, err := internal.NewComputeNode(ctx, computePort)
			if err != nil {
				return err
			}

			nodes = append(nodes, node)

			connectToMultiAddress := ""

			// if we started any ipfs servers already, use the first one
			if len(ipfsMultiAddresses) > 0 {
				connectToMultiAddress = ipfsMultiAddresses[0]
			}

			fmt.Printf("STARTING IPFS: %s\n", connectToMultiAddress)
			//ipfsRepo, ipfsMultiaddress, err := ipfs.StartBacalhauDevelopmentIpfsServer(connectToMultiAddress)
			_, ipfsMultiaddress, err := ipfs.StartBacalhauDevelopmentIpfsServer(connectToMultiAddress)
			if err != nil {
				return err
			}

			fmt.Printf("GOT ADDRESS: %s\n", ipfsMultiaddress)

			ipfsMultiAddresses = append(ipfsMultiAddresses, ipfsMultiaddress)

			// node.IpfsRepo = ipfsRepo

			// if i > 0 {
			// 	// connect to the first node
			// 	err = node.Connect(nodes[0].Host.Addrs()[0].String())
			// 	if err != nil {
			// 		return err
			// 	}
			// }

			// create a directory
		}

		// wait forever
		select {}

		return nil

	},
}
