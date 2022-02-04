package bacalhau

import (
	"context"
	"fmt"
	"log"

	"github.com/filecoin-project/bacalhau/internal"
	"github.com/phayes/freeport"
	"github.com/spf13/cobra"
)

var peerConnect string
var hostAddress string
var hostPort int

func init() {
	serveCmd.PersistentFlags().StringVar(
		&peerConnect, "peer", "",
		`The libp2p multiaddress to connect to.`,
	)
	serveCmd.PersistentFlags().StringVar(
		&hostAddress, "host", "127.0.0.1",
		`The port to listen on.`,
	)
	serveCmd.PersistentFlags().IntVar(
		&hostPort, "port", 0,
		`The port to listen on.`,
	)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the bacalhau compute node",
	RunE: func(cmd *cobra.Command, args []string) error {

		ctx := context.Background()

		computeNode, err := internal.NewComputeNode(ctx, hostPort)
		if err != nil {
			return err
		}
		err = computeNode.Connect(peerConnect)
		if err != nil {
			return err
		}

		jsonRpcString := ""
		devString := ""

		if developmentMode {
			jsonRpcPort, err := freeport.GetFreePort()
			if err != nil {
				log.Fatal(err)
			}
			jsonRpcString = fmt.Sprintf(" --jsonrpc-port %d", jsonRpcPort)
			devString = " --dev"
		}

		fmt.Printf(`
Command to connect other peers:

go run . serve --peer /ip4/%s/tcp/%d/p2p/%s%s%s
		
`, hostAddress, hostPort, computeNode.Host.ID(), jsonRpcString, devString)

		// run the jsonrpc server, passing it a reference to the pubsub topic so
		// that the CLI can also send messages to the chat room
		internal.RunBacalhauRpcServer(hostAddress, jsonrpcPort, computeNode)

		return nil

	},
}
