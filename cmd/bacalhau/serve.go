package bacalhau

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/compute_node"
	executor_util "github.com/filecoin-project/bacalhau/pkg/executor/util"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/requestor_node"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/libp2p"
	verifier_util "github.com/filecoin-project/bacalhau/pkg/verifier/util"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var peerConnect string
var ipfsConnect string
var hostAddress string
var hostPort int
var jobSelectionDataLocality string
var jobSelectionDataRejectStateless bool
var jobSelectionProbeHttp string
var jobSelectionProbeExec string

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
	serveCmd.PersistentFlags().StringVar(
		&jobSelectionDataLocality, "job-selection-data-locality", "local",
		`Only accept jobs that reference data we have locally ("local") or anywhere ("anywhere").`,
	)
	serveCmd.PersistentFlags().BoolVar(
		&jobSelectionDataRejectStateless, "job-selection-reject-stateless", false,
		`Reject jobs that don't specify any data.`,
	)
	serveCmd.PersistentFlags().StringVar(
		&jobSelectionProbeHttp, "job-selection-probe-http", "",
		`Use the result of a HTTP POST to decide if we should take on the job.`,
	)
	serveCmd.PersistentFlags().StringVar(
		&jobSelectionProbeExec, "job-selection-probe-exec", "",
		`Use the result of a exec an external program to decide if we should take on the job.`,
	)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the bacalhau compute node",
	RunE: func(cmd *cobra.Command, args []string) error {
		if ipfsConnect == "" {
			return fmt.Errorf("must specify ipfs-connect")
		}

		if jobSelectionDataLocality != "local" && jobSelectionDataLocality != "anywhere" {
			return fmt.Errorf("job-selection-data-locality must be either 'local' or 'anywhere'")
		}

		// Cleanup manager ensures that resources are freed before exiting:
		cm := system.NewCleanupManager()
		cm.RegisterCallback(system.CleanupTracer)
		defer cm.Cleanup()

		transport, err := libp2p.NewTransport(cm, hostPort)
		if err != nil {
			return err
		}

		requesterNode, err := requestor_node.NewRequesterNode(transport)
		if err != nil {
			return err
		}

		executors, err := executor_util.NewDockerIPFSExecutors(cm, ipfsConnect,
			fmt.Sprintf("bacalhau-%s", transport.Host.ID().String()))
		if err != nil {
			return err
		}

		verifiers, err := verifier_util.NewIPFSVerifiers(cm, ipfsConnect)
		if err != nil {
			return err
		}

		// construct the job selection policy from the CLI args
		typedJobSelectionDataLocality := compute_node.Local

		if jobSelectionDataLocality == "anywhere" {
			typedJobSelectionDataLocality = compute_node.Anywhere
		}

		jobSelectionPolicy := compute_node.JobSelectionPolicy{
			Locality:            typedJobSelectionDataLocality,
			RejectStatelessJobs: jobSelectionDataRejectStateless,
			ProbeHttp:           jobSelectionProbeHttp,
			ProbeExec:           jobSelectionProbeExec,
		}

		_, err = compute_node.NewComputeNode(
			transport,
			executors,
			verifiers,
			jobSelectionPolicy,
		)
		if err != nil {
			return err
		}

		apiServer := publicapi.NewServer(
			requesterNode,
			hostAddress,
			apiPort,
		)

		// Context ensures main goroutine waits until killed with ctrl+c:
		ctx, cancel := system.WithSignalShutdown(context.Background())
		defer cancel()

		go func(ctx context.Context) {
			if err := apiServer.ListenAndServe(ctx); err != nil {
				panic(err) // if api server can't run, bacalhau should stop
			}
		}(ctx)

		go func(ctx context.Context) {
			if err = transport.Start(ctx); err != nil {
				panic(err) // if transport can't run, bacalhau should stop
			}
		}(ctx)

		log.Debug().Msgf("libp2p server started: %d", hostPort)

		if peerConnect == "" {
			for _, addr := range DefaultBootstrapAddresses {
				err = transport.Connect(ctx, addr)
				if err != nil {
					return err
				}
			}
		} else if peerConnect != "none" {
			err = transport.Connect(ctx, peerConnect)
			if err != nil {
				return err
			}
			log.Debug().Msgf("libp2p connecting to: %s", peerConnect)
		}

		log.Info().Msgf("Bacalhau compute node started - peer id is: %s", transport.Host.ID().String())

		<-ctx.Done() // block until killed
		return nil
	},
}
