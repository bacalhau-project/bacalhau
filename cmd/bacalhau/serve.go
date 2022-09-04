package bacalhau

import (
	"context"
	"fmt"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/capacitymanager"
	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/libp2p"
	"github.com/filecoin-project/bacalhau/pkg/util/templates"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"
)

var DefaultBootstrapAddresses = []string{
	"/ip4/35.245.115.191/tcp/1235/p2p/QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
	"/ip4/35.245.61.251/tcp/1235/p2p/QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
	"/ip4/35.245.251.239/tcp/1235/p2p/QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3",
}
var DefaultSwarmPort = 1235

var (
	serveLong = templates.LongDesc(i18n.T(`
		Start the bacalhau campute node.
		`))

	serveExample = templates.Examples(i18n.T(`
		TBD`))

	OS = NewServeOptions()
)

type ServeOptions struct {
	PeerConnect                     string // The libp2p multiaddress to connect to.
	IPFSConnect                     string // The IPFS multiaddress to connect to.
	FilecoinUnsealedPath            string // The go template that can turn a filecoin CID into a local filepath with the unsealed data
	HostAddress                     string // The host address to listen on.
	SwarmPort                       int    // The host port for libp2p network.
	JobSelectionDataLocality        string // The data locality to use for job selection.
	JobSelectionDataRejectStateless bool   // Whether to reject jobs that don't specify any data.
	JobSelectionProbeHTTP           string // The HTTP URL to use for job selection.
	JobSelectionProbeExec           string // The executable to use for job selection.
	MetricsPort                     int    // The port to listen on for metrics.
	LimitTotalCPU                   string // The total amount of CPU the system can be using at one time.
	LimitTotalMemory                string // The total amount of memory the system can be using at one time.
	LimitTotalGPU                   string // The total amount of GPU the system can be using at one time.
	LimitJobCPU                     string // The amount of CPU the system can be using at one time for a single job.
	LimitJobMemory                  string // The amount of memory the system can be using at one time for a single job.
	LimitJobGPU                     string // The amount of GPU the system can be using at one time for a single job.
}

func NewServeOptions() *ServeOptions {
	return &ServeOptions{
		PeerConnect:                     "",
		IPFSConnect:                     "",
		FilecoinUnsealedPath:            "",
		HostAddress:                     "0.0.0.0",
		SwarmPort:                       DefaultSwarmPort,
		MetricsPort:                     2112,
		JobSelectionDataLocality:        "local",
		JobSelectionDataRejectStateless: false,
		JobSelectionProbeHTTP:           "",
		JobSelectionProbeExec:           "",
		LimitTotalCPU:                   "",
		LimitTotalMemory:                "",
		LimitTotalGPU:                   "",
		LimitJobCPU:                     "",
		LimitJobMemory:                  "",
		LimitJobGPU:                     "",
	}
}

func setupJobSelectionCLIFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(
		&OS.JobSelectionDataLocality, "job-selection-data-locality", OS.JobSelectionDataLocality,
		`Only accept jobs that reference data we have locally ("local") or anywhere ("anywhere").`,
	)
	cmd.PersistentFlags().BoolVar(
		&OS.JobSelectionDataRejectStateless, "job-selection-reject-stateless", OS.JobSelectionDataRejectStateless,
		`Reject jobs that don't specify any data.`,
	)
	cmd.PersistentFlags().StringVar(
		&OS.JobSelectionProbeHTTP, "job-selection-probe-http", OS.JobSelectionProbeHTTP,
		`Use the result of a HTTP POST to decide if we should take on the job.`,
	)
	cmd.PersistentFlags().StringVar(
		&OS.JobSelectionProbeExec, "job-selection-probe-exec", OS.JobSelectionProbeExec,
		`Use the result of a exec an external program to decide if we should take on the job.`,
	)
}

func setupCapacityManagerCLIFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(
		&OS.LimitTotalCPU, "limit-total-cpu", OS.LimitTotalCPU,
		`Total CPU core limit to run all jobs (e.g. 500m, 2, 8).`,
	)
	cmd.PersistentFlags().StringVar(
		&OS.LimitTotalMemory, "limit-total-memory", OS.LimitTotalMemory,
		`Total Memory limit to run all jobs  (e.g. 500Mb, 2Gb, 8Gb).`,
	)
	cmd.PersistentFlags().StringVar(
		&OS.LimitTotalGPU, "limit-total-gpu", OS.LimitTotalGPU,
		`Total GPU limit to run all jobs (e.g. 1, 2, or 8).`,
	)
	cmd.PersistentFlags().StringVar(
		&OS.LimitJobCPU, "limit-job-cpu", OS.LimitJobCPU,
		`Job CPU core limit for single job (e.g. 500m, 2, 8).`,
	)
	cmd.PersistentFlags().StringVar(
		&OS.LimitJobMemory, "limit-job-memory", OS.LimitJobMemory,
		`Job Memory limit for single job  (e.g. 500Mb, 2Gb, 8Gb).`,
	)
	cmd.PersistentFlags().StringVar(
		&OS.LimitJobGPU, "limit-job-gpu", OS.LimitJobGPU,
		`Job GPU limit for single job (e.g. 1, 2, or 8).`,
	)
}

func getPeers() []string {
	if OS.PeerConnect == "none" {
		return []string{}
	} else if OS.PeerConnect == "" {
		return DefaultBootstrapAddresses
	}
	return strings.Split(OS.PeerConnect, ",")
}

func getJobSelectionConfig() computenode.JobSelectionPolicy {
	// construct the job selection policy from the CLI args
	typedJobSelectionDataLocality := computenode.Anywhere

	if OS.JobSelectionDataLocality == "anywhere" {
		typedJobSelectionDataLocality = computenode.Anywhere
	}

	jobSelectionPolicy := computenode.JobSelectionPolicy{
		Locality:            typedJobSelectionDataLocality,
		RejectStatelessJobs: OS.JobSelectionDataRejectStateless,
		ProbeHTTP:           OS.JobSelectionProbeHTTP,
		ProbeExec:           OS.JobSelectionProbeExec,
	}

	return jobSelectionPolicy
}

func getCapacityManagerConfig() capacitymanager.Config {
	// the total amount of CPU / Memory the system can be using at one time
	totalResourceLimit := model.ResourceUsageConfig{
		CPU:    OS.LimitTotalCPU,
		Memory: OS.LimitTotalMemory,
		GPU:    OS.LimitTotalGPU,
	}

	// the per job CPU / Memory limits
	jobResourceLimit := model.ResourceUsageConfig{
		CPU:    OS.LimitJobCPU,
		Memory: OS.LimitJobMemory,
		GPU:    OS.LimitJobGPU,
	}

	return capacitymanager.Config{
		ResourceLimitTotal: totalResourceLimit,
		ResourceLimitJob:   jobResourceLimit,
	}
}

func init() { //nolint:gochecknoinits // Using init in cobra command is idomatic
	serveCmd.PersistentFlags().StringVar(
		&OS.PeerConnect, "peer", OS.PeerConnect,
		`The libp2p multiaddress to connect to.`,
	)
	serveCmd.PersistentFlags().StringVar(
		&OS.IPFSConnect, "ipfs-connect", OS.IPFSConnect,
		`The ipfs host multiaddress to connect to.`,
	)
	serveCmd.PersistentFlags().StringVar(
		&OS.FilecoinUnsealedPath, "filecoin-unsealed-path", OS.FilecoinUnsealedPath,
		`The go template that can turn a filecoin CID into a local filepath with the unsealed data.`,
	)
	serveCmd.PersistentFlags().StringVar(
		&OS.HostAddress, "host", OS.HostAddress,
		`The host to listen on (for both api and swarm connections).`,
	)
	serveCmd.PersistentFlags().IntVar(
		&OS.SwarmPort, "swarm-port", OS.SwarmPort,
		`The port to listen on for swarm connections.`,
	)
	serveCmd.PersistentFlags().IntVar(
		&OS.MetricsPort, "metrics-port", OS.MetricsPort,
		`The port to serve prometheus metrics on.`,
	)

	setupJobSelectionCLIFlags(serveCmd)
	setupCapacityManagerCLIFlags(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:     "serve",
	Short:   "Start the bacalhau compute node",
	Long:    serveLong,
	Example: serveExample,
	RunE: func(cmd *cobra.Command, args []string) error {
		cm := system.NewCleanupManager()
		cm.RegisterCallback(system.CleanupTraceProvider)
		defer cm.Cleanup()

		ctx := cmd.Context()
		t := system.GetTracer()
		ctx, rootSpan := system.NewRootSpan(ctx, t, "cmd/bacalhau/serve")
		defer rootSpan.End()
		cm.RegisterCallback(system.CleanupTraceProvider)

		if OS.IPFSConnect == "" {
			return fmt.Errorf("must specify ipfs-connect")
		}

		if OS.JobSelectionDataLocality != "local" && OS.JobSelectionDataLocality != "anywhere" {
			return fmt.Errorf("job-selection-data-locality must be either 'local' or 'anywhere'")
		}

		// Cleanup manager ensures that resources are freed before exiting:
		cm := system.NewCleanupManager()
		cm.RegisterCallback(system.CleanupTraceProvider)
		defer cm.Cleanup()

		ctx := context.Background()

		peers := DefaultBootstrapAddresses // Default to connecting to defaults
		if OS.PeerConnect == "none" {
			peers = []string{} // Only connect to peers if not none
		} else if OS.PeerConnect != "" {
			peers = []string{OS.PeerConnect} // Otherwise set peers according to the user options
		}

		log.Debug().Msgf("libp2p connecting to: %s", strings.Join(peers, ", "))

		datastore, err := inmemory.NewInMemoryDatastore()
		if err != nil {
			return err
		}

		transport, err := libp2p.NewTransport(ctx, cm, OS.HostPort, peers)
		if err != nil {
			return err
		}

		hostID, err := transport.HostID(ctx)
		if err != nil {
			return err
		}

		storageProviders, err := executor_util.NewStandardStorageProviders(
			ctx,
			cm,
			executor_util.StandardStorageProviderOptions{
				IPFSMultiaddress:     OS.IPFSConnect,
				FilecoinUnsealedPath: OS.FilecoinUnsealedPath,
			},
		)
		if err != nil {
			return err
		}

		controller, err := controller.NewController(
			ctx,
			cm,
			datastore,
			transport,
			storageProviders,
		)
		if err != nil {
			return err
		}

		executors, err := executor_util.NewStandardExecutors(
			ctx,
			cm,
			executor_util.StandardExecutorOptions{
				DockerID: fmt.Sprintf("bacalhau-%s", hostID),
				Storage: executor_util.StandardStorageProviderOptions{
					IPFSMultiaddress:     OS.IPFSConnect,
					FilecoinUnsealedPath: OS.FilecoinUnsealedPath,
				},
			},
		)
		if err != nil {
			return err
		}

		verifiers, err := verifier_util.NewStandardVerifiers(
			ctx,
			cm,
			controller.GetStateResolver(),
			transport.Encrypt,
			transport.Decrypt,
		)
		if err != nil {
			return err
		}

		publishers, err := publisher_util.NewIPFSPublishers(ctx, cm, controller.GetStateResolver(), OS.IPFSConnect)
		if err != nil {
			return err
		}

		jobSelectionPolicy := getJobSelectionConfig()
		totalResourceLimit, jobResourceLimit := getCapacityManagerConfig()

		computeNodeConfig := computenode.ComputeNodeConfig{
			JobSelectionPolicy: jobSelectionPolicy,
			CapacityManagerConfig: capacitymanager.Config{
				ResourceLimitTotal: totalResourceLimit,
				ResourceLimitJob:   jobResourceLimit,
			},
		}

		requesterNodeConfig := requesternode.RequesterNodeConfig{}

		_, err = requesternode.NewRequesterNode(
			ctx,
			cm,
			controller,
			verifiers,
			requesterNodeConfig,
		)
		if err != nil {
			return err
		}
		_, err = computenode.NewComputeNode(
			ctx,
			cm,
			controller,
			executors,
			verifiers,
			publishers,
			computeNodeConfig,
		)
		if err != nil {
			return err
		}

		apiServer := publicapi.NewServer(
			ctx,
			OS.HostAddress,
			apiPort,
			controller,
			publishers,
		)

		// Context ensures main goroutine waits until killed with ctrl+c:
		ctx, cancel := system.WithSignalShutdown(ctx)
		defer cancel()

		go func(ctx context.Context) {
			if err = apiServer.ListenAndServe(ctx, cm); err != nil {
				log.Fatal().Msgf("Api server can't run, bacalhau should stop: %+v", err)
			}
		}(ctx)

		go func(ctx context.Context) {
			if err = controller.Start(ctx); err != nil {
				log.Fatal().Msgf("Controller can't run, bacalhau should stop: %+v", err)
			}
			if err = transport.Start(ctx); err != nil {
				log.Fatal().Msgf("Transport can't run, bacalhau should stop: %+v", err)
			}
		}(ctx)

		go func(ctx context.Context) { //nolint:unparam // ctx appropriate here
			if err = system.ListenAndServeMetrics(ctx, cm, OS.MetricsPort); err != nil {
				log.Error().Msgf("Cannot serve metrics: %v", err)
			}
		}(ctx)

		log.Debug().Msgf("libp2p server started: %d", OS.HostPort)

		log.Info().Msgf("Bacalhau compute node started - peer id is: %s", hostID)

		<-ctx.Done() // block until killed
		return nil
	},
}
