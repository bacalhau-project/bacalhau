package bacalhau

import (
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/compute_node"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/requestor_node"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/libp2p"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var peerConnect string
var ipfsConnect string
var hostAddress string
var hostPort int

var DefaultBootstrapAddresses = []string{
	"/ip4/35.245.115.191/tcp/1235/p2p/QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
	"/ip4/35.245.61.251/tcp/1235/p2p/QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
	"/ip4/35.245.251.239/tcp/1235/p2p/QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3",
}

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
		`The host to listen on (for both api and swarm connections).`,
	)
	serveCmd.PersistentFlags().IntVar(
		&hostPort, "port", 1235,
		`The port to listen on for swarm connections.`,
	)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the bacalhau compute node",
	RunE: func(cmd *cobra.Command, args []string) error { // nolint

		if ipfsConnect == "" {
			return fmt.Errorf("must specify ipfs-connect")
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
		if err != nil {
			return err
		}

		verifiers, err := verifier.NewIPFSVerifiers(cancelContext, ipfsConnect)
		if err != nil {
			return err
		}

		_, err = compute_node.NewComputeNode(transport, executors, verifiers)
		if err != nil {
			return err
		}

		apiServer := publicapi.NewServer(
			requesterNode,
			apiHost,
			apiPort,
		)

		go func() {
			if err := apiServer.ListenAndServe(cancelContext); err != nil {
				panic(err) // if api server can't run, bacalhau should stop
			}
		}()

		err = transport.Start()
		if err != nil {
			return err
		}

		log.Debug().Msgf("libp2p server started: %d", hostPort)

		if peerConnect == "" {
			for _, addr := range DefaultBootstrapAddresses {
				err = transport.Connect(addr)
				if err != nil {
					return err
				}
			}
		} else {
			err = transport.Connect(peerConnect)
			if err != nil {
				return err
			}
			log.Debug().Msgf("libp2p connecting to: %s", peerConnect)
		}

		log.Info().Msgf("Bacalhau compute node started - peer id is: %s", transport.Host.ID().String())

		select {}
	},
}
