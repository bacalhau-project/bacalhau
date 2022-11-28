package bacalhau

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/compute/capacity"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	filecoinlotus "github.com/filecoin-project/bacalhau/pkg/publisher/filecoin_lotus"

	"github.com/filecoin-project/bacalhau/pkg/localdb/inmemory"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/libp2p"
	"github.com/filecoin-project/bacalhau/pkg/util/templates"
	"github.com/multiformats/go-multiaddr"

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
	PeerConnect                     string        // The libp2p multiaddress to connect to.
	IPFSConnect                     string        // The IPFS multiaddress to connect to.
	FilecoinUnsealedPath            string        // The go template that can turn a filecoin CID into a local filepath with the unsealed data.
	EstuaryAPIKey                   string        // The API key used when using the estuary API.
	HostAddress                     string        // The host address to listen on.
	SwarmPort                       int           // The host port for libp2p network.
	JobSelectionDataLocality        string        // The data locality to use for job selection.
	JobSelectionDataRejectStateless bool          // Whether to reject jobs that don't specify any data.
	JobSelectionProbeHTTP           string        // The HTTP URL to use for job selection.
	JobSelectionProbeExec           string        // The executable to use for job selection.
	MetricsPort                     int           // The port to listen on for metrics.
	LimitTotalCPU                   string        // The total amount of CPU the system can be using at one time.
	LimitTotalMemory                string        // The total amount of memory the system can be using at one time.
	LimitTotalGPU                   string        // The total amount of GPU the system can be using at one time.
	LimitJobCPU                     string        // The amount of CPU the system can be using at one time for a single job.
	LimitJobMemory                  string        // The amount of memory the system can be using at one time for a single job.
	LimitJobGPU                     string        // The amount of GPU the system can be using at one time for a single job.
	LotusFilecoinStorageDuration    time.Duration // How long deals should be for the Lotus Filecoin publisher
	LotusFilecoinPathDirectory      string        // The location of the Lotus configuration directory which contains config.toml, etc
	LotusFilecoinUploadDirectory    string        // Directory to put files when uploading to Lotus (optional)
	LotusFilecoinMaximumPing        time.Duration // The maximum ping allowed when selecting a Filecoin miner
}

func NewServeOptions() *ServeOptions {
	return &ServeOptions{
		PeerConnect:                     "",
		IPFSConnect:                     "",
		FilecoinUnsealedPath:            "",
		EstuaryAPIKey:                   os.Getenv("ESTUARY_API_KEY"),
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
		LotusFilecoinPathDirectory:      os.Getenv("LOTUS_PATH"),
		LotusFilecoinMaximumPing:        2 * time.Second,
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

func setupLibp2pCLIFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(
		&OS.PeerConnect, "peer", OS.PeerConnect,
		`The libp2p multiaddress to connect to.`,
	)
	cmd.PersistentFlags().StringVar(
		&OS.HostAddress, "host", OS.HostAddress,
		`The host to listen on (for both api and swarm connections).`,
	)
	cmd.PersistentFlags().IntVar(
		&OS.SwarmPort, "swarm-port", OS.SwarmPort,
		`The port to listen on for swarm connections.`,
	)
}

func getPeers() []multiaddr.Multiaddr {
	var peersStrings []string
	if OS.PeerConnect == "none" {
		peersStrings = []string{}
	} else if OS.PeerConnect == "" {
		peersStrings = DefaultBootstrapAddresses
	} else {
		peersStrings = strings.Split(OS.PeerConnect, ",")
	}
	// convert peers stringsto multiaddrs
	peers := make([]multiaddr.Multiaddr, len(peersStrings))
	for i, peer := range peersStrings {
		peers[i], _ = multiaddr.NewMultiaddr(peer)
	}
	return peers
}

func getJobSelectionConfig() model.JobSelectionPolicy {
	// construct the job selection policy from the CLI args
	typedJobSelectionDataLocality := model.Anywhere

	if OS.JobSelectionDataLocality == "anywhere" {
		typedJobSelectionDataLocality = model.Anywhere
	}

	jobSelectionPolicy := model.JobSelectionPolicy{
		Locality:            typedJobSelectionDataLocality,
		RejectStatelessJobs: OS.JobSelectionDataRejectStateless,
		ProbeHTTP:           OS.JobSelectionProbeHTTP,
		ProbeExec:           OS.JobSelectionProbeExec,
	}

	return jobSelectionPolicy
}

func getComputeConfig() node.ComputeConfig {
	return node.NewComputeConfigWith(node.ComputeConfigParams{
		JobSelectionPolicy: getJobSelectionConfig(),
		TotalResourceLimits: capacity.ParseResourceUsageConfig(model.ResourceUsageConfig{
			CPU:    OS.LimitTotalCPU,
			Memory: OS.LimitTotalMemory,
			GPU:    OS.LimitTotalGPU,
		}),
		JobResourceLimits: capacity.ParseResourceUsageConfig(model.ResourceUsageConfig{
			CPU:    OS.LimitJobCPU,
			Memory: OS.LimitJobMemory,
			GPU:    OS.LimitJobGPU,
		}),
		IgnorePhysicalResourceLimits: os.Getenv("BACALHAU_CAPACITY_MANAGER_OVER_COMMIT") != "",
	})
}

func init() { //nolint:gochecknoinits // Using init in cobra command is idomatic
	serveCmd.PersistentFlags().StringVar(
		&OS.IPFSConnect, "ipfs-connect", OS.IPFSConnect,
		`The ipfs host multiaddress to connect to.`,
	)
	serveCmd.PersistentFlags().StringVar(
		&OS.FilecoinUnsealedPath, "filecoin-unsealed-path", OS.FilecoinUnsealedPath,
		`The go template that can turn a filecoin CID into a local filepath with the unsealed data.`,
	)
	serveCmd.PersistentFlags().StringVar(
		&OS.EstuaryAPIKey, "estuary-api-key", OS.EstuaryAPIKey,
		`The API key used when using the estuary API.`,
	)
	serveCmd.PersistentFlags().IntVar(
		&OS.MetricsPort, "metrics-port", OS.MetricsPort,
		`The port to serve prometheus metrics on.`,
	)
	serveCmd.PersistentFlags().DurationVar(
		&OS.LotusFilecoinStorageDuration, "lotus-storage-duration", OS.LotusFilecoinStorageDuration,
		"Duration to store data in Lotus Filecoin for.",
	)
	serveCmd.PersistentFlags().StringVar(
		&OS.LotusFilecoinPathDirectory, "lotus-path-directory", OS.LotusFilecoinPathDirectory,
		"Location of the Lotus Filecoin configuration directory.",
	)
	serveCmd.PersistentFlags().StringVar(
		&OS.LotusFilecoinUploadDirectory, "lotus-upload-directory", OS.LotusFilecoinUploadDirectory,
		"Directory to use when uploading content to Lotus Filecoin.",
	)
	serveCmd.PersistentFlags().DurationVar(
		&OS.LotusFilecoinMaximumPing, "lotus-max-ping", OS.LotusFilecoinMaximumPing,
		"The highest ping a Filecoin miner could have when selecting.",
	)

	setupLibp2pCLIFlags(serveCmd)
	setupJobSelectionCLIFlags(serveCmd)
	setupCapacityManagerCLIFlags(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:     "serve",
	Short:   "Start the bacalhau compute node",
	Long:    serveLong,
	Example: serveExample,
	RunE: func(cmd *cobra.Command, _ []string) error {
		// Cleanup manager ensures that resources are freed before exiting:
		cm := system.NewCleanupManager()
		cm.RegisterCallback(system.CleanupTraceProvider)
		defer cm.Cleanup()

		ctx := cmd.Context()

		// Context ensures main goroutine waits until killed with ctrl+c:
		ctx, cancel := system.WithSignalShutdown(ctx)
		defer cancel()

		ctx, rootSpan := system.NewRootSpan(ctx, system.GetTracer(), "cmd/bacalhau/serve")
		defer rootSpan.End()
		cm.RegisterCallback(system.CleanupTraceProvider)

		if OS.IPFSConnect == "" {
			Fatal("You must specify --ipfs-connect.", 1)
		}

		if OS.JobSelectionDataLocality != "local" && OS.JobSelectionDataLocality != "anywhere" {
			Fatal("--job-selection-data-locality must be either 'local' or 'anywhere'", 1)
		}

		// Establishing p2p connection
		peers := getPeers()
		log.Debug().Msgf("libp2p connecting to: %s", peers)

		transport, err := libp2p.NewTransport(ctx, cm, OS.SwarmPort, peers)
		if err != nil {
			Fatal(fmt.Sprintf("Error creating libp2p transport: %s", err), 1)
		}

		// add nodeID to logging context
		ctx = logger.ContextWithNodeIDLogger(ctx, transport.HostID())

		// Establishing IPFS connection
		ipfs, err := ipfs.NewClient(OS.IPFSConnect)
		if err != nil {
			Fatal(fmt.Sprintf("Error creating IPFS client: %s", err), 1)
		}

		datastore, err := inmemory.NewInMemoryDatastore()
		if err != nil {
			Fatal(fmt.Sprintf("Error creating in memory datastore: %s", err), 1)
		}

		// Create node config from cmd arguments
		nodeConfig := node.NodeConfig{
			IPFSClient:           ipfs,
			CleanupManager:       cm,
			LocalDB:              datastore,
			Transport:            transport,
			FilecoinUnsealedPath: OS.FilecoinUnsealedPath,
			EstuaryAPIKey:        OS.EstuaryAPIKey,
			HostAddress:          OS.HostAddress,
			APIPort:              apiPort,
			MetricsPort:          OS.MetricsPort,
			ComputeConfig:        getComputeConfig(),
			RequesterNodeConfig:  requesternode.NewDefaultRequesterNodeConfig(),
		}

		if OS.LotusFilecoinStorageDuration != time.Duration(0) &&
			OS.LotusFilecoinPathDirectory != "" &&
			OS.LotusFilecoinMaximumPing != time.Duration(0) {
			nodeConfig.LotusConfig = &filecoinlotus.PublisherConfig{
				StorageDuration: OS.LotusFilecoinStorageDuration,
				PathDir:         OS.LotusFilecoinPathDirectory,
				UploadDir:       OS.LotusFilecoinUploadDirectory,
				MaximumPing:     OS.LotusFilecoinMaximumPing,
			}
		}

		// Create node
		node, err := node.NewStandardNode(ctx, nodeConfig)
		if err != nil {
			Fatal(fmt.Sprintf("Error creating node: %s", err), 1)
		}

		// Start transport layer
		err = transport.Start(ctx)
		if err != nil {
			Fatal(fmt.Sprintf("Error starting transport layer: %s", err), 1)
		}

		// Start node
		err = node.Start(ctx)
		if err != nil {
			Fatal(fmt.Sprintf("Error starting node: %s", err), 1)
		}

		<-ctx.Done() // block until killed
		return nil
	},
}
