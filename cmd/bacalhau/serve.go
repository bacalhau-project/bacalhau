package bacalhau

import (
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/compute_node"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/jsonrpc"
	"github.com/filecoin-project/bacalhau/pkg/requestor_node"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/libp2p"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var peerConnect string
var ipfsConnect string
var hostAddress string
var hostPort int

func init() {
	serveCmd.PersistentFlags().StringVar(
		&peerConnect, "peer", "",
		`The libp2p multiaddress to connect to.`,
	)
	serveCmd.PersistentFlags().StringVar(
		&ipfsConnect, "ipfs-connect", "",
		`The ipfs host multiaddress to connect to.`,
	)
	serveCmd.PersistentFlags().StringVar(
		&hostAddress, "host", "0.0.0.0",
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
	RunE: func(cmd *cobra.Command, args []string) error { // nolint

		if ipfsConnect == "" {
			return fmt.Errorf("Must specify ipfs-connect")
		}

		cancelContext := system.GetCancelContext()
		transport, err := libp2p.NewLibp2pTransport(cancelContext, hostPort)
		if err != nil {
			return err
		}

		requesterNode, err := requestor_node.NewRequesterNode(transport)
		if err != nil {
			return err
		}

		executors, err := executor.NewDockerIPFSExecutors(cancelContext, ipfsConnect, fmt.Sprintf("bacalhau-%s", transport.Host.ID().String()))

		_, err = compute_node.NewComputeNode(transport, executors)
		if err != nil {
			return err
		}

		jsonRpcNode := jsonrpc.NewBacalhauJsonRpcServer(
			cancelContext,
			jsonrpcHost,
			jsonrpcPort,
			requesterNode,
		)
		if err != nil {
			return err
		}

		err = jsonrpc.StartBacalhauJsonRpcServer(jsonRpcNode)
		if err != nil {
			return err
		}

		log.Info().Msgf("Bacalhau compute node started - peer id is: %s", transport.Host.ID().String())

		select {}
	},
}
