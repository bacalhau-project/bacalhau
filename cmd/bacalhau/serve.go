package bacalhau

import (
	"context"
	"fmt"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/capacitymanager"
	computenode "github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/controller"
	executor_util "github.com/filecoin-project/bacalhau/pkg/executor/util"
	"github.com/filecoin-project/bacalhau/pkg/localdb/inmemory"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"
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
var jobSelectionProbeHTTP string
var jobSelectionProbeExec string
var metricsPort = 2112
var limitTotalCPU string
var limitTotalMemory string
var limitTotalGPU string
var limitJobCPU string
var limitJobMemory string
var limitJobGPU string

var DefaultBootstrapAddresses = []string{
	"/ip4/35.245.115.191/tcp/1235/p2p/QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
	"/ip4/35.245.61.251/tcp/1235/p2p/QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
	"/ip4/35.245.251.239/tcp/1235/p2p/QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3",
}
var DefaultSwarmPort = 1235

func init() { // nolint:gochecknoinits // Using init in cobra command is idomatic
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
		&hostPort, "port", DefaultSwarmPort,
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
		&jobSelectionProbeHTTP, "job-selection-probe-http", "",
		`Use the result of a HTTP POST to decide if we should take on the job.`,
	)
	serveCmd.PersistentFlags().StringVar(
		&jobSelectionProbeExec, "job-selection-probe-exec", "",
		`Use the result of a exec an external program to decide if we should take on the job.`,
	)
	serveCmd.PersistentFlags().IntVar(
		&metricsPort, "metrics-port", metricsPort,
		`The port to serve prometheus metrics on.`,
	)
	serveCmd.PersistentFlags().StringVar(
		&limitTotalCPU, "limit-total-cpu", "",
		`Total CPU core limit to run all jobs (e.g. 500m, 2, 8).`,
	)
	serveCmd.PersistentFlags().StringVar(
		&limitTotalMemory, "limit-total-memory", "",
		`Total Memory limit to run all jobs  (e.g. 500Mb, 2Gb, 8Gb).`,
	)
	serveCmd.PersistentFlags().StringVar(
		&limitTotalGPU, "limit-total-gpu", "",
		`Total GPU limit to run all jobs (e.g. 1, 2, or 8).`,
	)
	serveCmd.PersistentFlags().StringVar(
		&limitJobCPU, "limit-job-cpu", "",
		`Job CPU core limit for single job (e.g. 500m, 2, 8).`,
	)
	serveCmd.PersistentFlags().StringVar(
		&limitJobMemory, "limit-job-memory", "",
		`Job Memory limit for single job  (e.g. 500Mb, 2Gb, 8Gb).`,
	)
	serveCmd.PersistentFlags().StringVar(
		&limitJobGPU, "limit-job-gpu", "",
		`Job GPU limit for single job (e.g. 1, 2, or 8).`,
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

		peers := DefaultBootstrapAddresses

		if peerConnect != "" && peerConnect != "none" {
			peers = []string{peerConnect}
		}

		log.Debug().Msgf("libp2p connecting to: %s", strings.Join(peers, ", "))

		datastore, err := inmemory.NewInMemoryDatastore()
		if err != nil {
			return err
		}

		transport, err := libp2p.NewTransport(cm, hostPort, peers)
		if err != nil {
			return err
		}

		controller, err := controller.NewController(
			cm,
			datastore,
			transport,
		)
		if err != nil {
			return err
		}

		hostID, err := transport.HostID(context.Background())
		if err != nil {
			return err
		}
		executors, err := executor_util.NewStandardExecutors(
			cm,
			ipfsConnect,
			fmt.Sprintf("bacalhau-%s", hostID),
		)
		if err != nil {
			return err
		}

		verifiers, err := verifier_util.NewIPFSVerifiers(cm, ipfsConnect)
		if err != nil {
			return err
		}

		// construct the job selection policy from the CLI args
		typedJobSelectionDataLocality := computenode.Local

		if jobSelectionDataLocality == "anywhere" {
			typedJobSelectionDataLocality = computenode.Anywhere
		}

		// the total amount of CPU / Memory the system can be using at one time
		totalResourceLimit := capacitymanager.ResourceUsageConfig{
			CPU:    limitTotalCPU,
			Memory: limitTotalMemory,
			GPU:    limitTotalGPU,
		}

		// the per job CPU / Memory limits
		jobResourceLimit := capacitymanager.ResourceUsageConfig{
			CPU:    limitJobCPU,
			Memory: limitJobMemory,
			GPU:    limitJobGPU,
		}

		jobSelectionPolicy := computenode.JobSelectionPolicy{
			Locality:            typedJobSelectionDataLocality,
			RejectStatelessJobs: jobSelectionDataRejectStateless,
			ProbeHTTP:           jobSelectionProbeHTTP,
			ProbeExec:           jobSelectionProbeExec,
		}

		requesterNodeConfig := requesternode.RequesterNodeConfig{}

		computeNodeConfig := computenode.ComputeNodeConfig{
			JobSelectionPolicy: jobSelectionPolicy,
			CapacityManagerConfig: capacitymanager.Config{
				ResourceLimitTotal: totalResourceLimit,
				ResourceLimitJob:   jobResourceLimit,
			},
		}

		requesterNode, err := requesternode.NewRequesterNode(
			cm,
			controller,
			verifiers,
			requesterNodeConfig,
		)
		if err != nil {
			return err
		}
		_, err = computenode.NewComputeNode(
			cm,
			controller,
			executors,
			verifiers,
			computeNodeConfig,
		)
		if err != nil {
			return err
		}

		apiServer := publicapi.NewServer(
			hostAddress,
			apiPort,
			transport,
		)

		// Context ensures main goroutine waits until killed with ctrl+c:
		ctx, cancel := system.WithSignalShutdown(context.Background())
		defer cancel()

		go func(ctx context.Context) {
			if err = apiServer.ListenAndServe(ctx, cm); err != nil {
				log.Fatal().Msgf("Api server can't run, bacalhau should stop: %+v", err)
			}
		}(ctx)

		go func(ctx context.Context) {
			if err = transport.Start(ctx); err != nil {
				log.Fatal().Msgf("Transport can't run, bacalhau should stop: %+v", err)
			}
		}(ctx)

		// TODO: #352 should system.ListenAndServeMetrix take ctx?
		go func(ctx context.Context) { // nolint:unparam // ctx appropriate here
			if err = system.ListenAndServeMetrics(cm, metricsPort); err != nil {
				log.Error().Msgf("Cannot serve metrics: %v", err)
			}
		}(ctx)

		log.Debug().Msgf("libp2p server started: %d", hostPort)

		log.Info().Msgf("Bacalhau compute node started - peer id is: %s", hostID)

		<-ctx.Done() // block until killed
		return nil
	},
}
