package main

import (
	"context"

	"github.com/spf13/cobra"
)

var peerConnect string
var hostPort int

func init() {
	serveCmd.PersistentFlags().StringVar(
		&peerConnect, "peer", "",
		`The libp2p multiaddress to connect to.`,
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

		computeNode, err := NewComputeNode(ctx, hostPort)
		if err != nil {
			return err
		}
		err = computeNode.Connect(peerConnect)
		if err != nil {
			return err
		}

		// run the jsonrpc server, passing it a reference to the pubsub topic so
		// that the CLI can also send messages to the chat room
		go runBacalhauRpcServer(jsonrpcPort, computeNode)

		computeNode.Render()

		return nil

	},
}
