package bacalhau

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/compute/capacity"
	"github.com/filecoin-project/bacalhau/pkg/libp2p"
	"github.com/filecoin-project/bacalhau/pkg/libp2p/rcmgr"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	filecoinlotus "github.com/filecoin-project/bacalhau/pkg/publisher/filecoin_lotus"

	"github.com/filecoin-project/bacalhau/pkg/localdb/inmemory"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/templates"
	"github.com/multiformats/go-multiaddr"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"
)

var DefaultSwarmPort = 1235

var (
	serveLong = templates.LongDesc(i18n.T(`
		Start a bacalhau node.
		`))

	serveExample = templates.Examples(i18n.T(`
		# Start a bacalhau compute node
		bacalhau serve
		# or
		bacalhau serve --node-type compute

		# Start a bacalhau requester node
		bacalhau serve --node-type requester

		# Start a bacalhau hybrid node that acts as both compute and requester
		bacalhau serve --node-type compute --node-type requester
		# or
		bacalhau serve --node-type compute,requester
`))
)

//nolint:lll // Documentation
type ServeOptions struct {
	NodeType                              []string          // "compute", "requester" node or both
	PeerConnect                           string            // The libp2p multiaddress to connect to.
	IPFSConnect                           string            // The IPFS multiaddress to connect to.
	FilecoinUnsealedPath                  string            // Go template to turn a Filecoin CID into a local filepath with the unsealed data.
	EstuaryAPIKey                         string            // The API key used when using the estuary API.
	HostAddress                           string            // The host address to listen on.
	SwarmPort                             int               // The host port for libp2p network.
	JobSelectionDataLocality              string            // The data locality to use for job selection.
	JobSelectionDataRejectStateless       bool              // Whether to reject jobs that don't specify any data.
	JobSelectionDataAcceptNetworked       bool              // Whether to accept jobs that require network access.
	JobSelectionProbeHTTP                 string            // The HTTP URL to use for job selection.
	JobSelectionProbeExec                 string            // The executable to use for job selection.
	MetricsPort                           int               // The port to listen on for metrics.
	LimitTotalCPU                         string            // The total amount of CPU the system can be using at one time.
	LimitTotalMemory                      string            // The total amount of memory the system can be using at one time.
	LimitTotalGPU                         string            // The total amount of GPU the system can be using at one time.
	LimitJobCPU                           string            // The amount of CPU the system can be using at one time for a single job.
	LimitJobMemory                        string            // The amount of memory the system can be using at one time for a single job.
	LimitJobGPU                           string            // The amount of GPU the system can be using at one time for a single job.
	LotusFilecoinStorageDuration          time.Duration     // How long deals should be for the Lotus Filecoin publisher
	LotusFilecoinPathDirectory            string            // The location of the Lotus configuration directory which contains config.toml, etc
	LotusFilecoinUploadDirectory          string            // Directory to put files when uploading to Lotus (optional)
	LotusFilecoinMaximumPing              time.Duration     // The maximum ping allowed when selecting a Filecoin miner
	JobExecutionTimeoutClientIDBypassList []string          // IDs of clients that can submit jobs more than the configured job execution timeout
	Labels                                map[string]string // Labels to apply to the node that can be used for node selection and filtering
}

func NewServeOptions() *ServeOptions {
	return &ServeOptions{
		NodeType:                        []string{"compute"},
		PeerConnect:                     "",
		IPFSConnect:                     "",
		FilecoinUnsealedPath:            "",
		EstuaryAPIKey:                   os.Getenv("ESTUARY_API_KEY"),
		HostAddress:                     "0.0.0.0",
		SwarmPort:                       DefaultSwarmPort,
		MetricsPort:                     2112,
		JobSelectionDataLocality:        "local",
		JobSelectionDataRejectStateless: false,
		JobSelectionDataAcceptNetworked: false,
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

func setupJobSelectionCLIFlags(cmd *cobra.Command, OS *ServeOptions) {
	cmd.PersistentFlags().StringVar(
		&OS.JobSelectionDataLocality, "job-selection-data-locality", OS.JobSelectionDataLocality,
		`Only accept jobs that reference data we have locally ("local") or anywhere ("anywhere").`,
	)
	cmd.PersistentFlags().BoolVar(
		&OS.JobSelectionDataRejectStateless, "job-selection-reject-stateless", OS.JobSelectionDataRejectStateless,
		`Reject jobs that don't specify any data.`,
	)
	cmd.PersistentFlags().BoolVar(
		&OS.JobSelectionDataAcceptNetworked, "job-selection-accept-networked", OS.JobSelectionDataAcceptNetworked,
		`Accept jobs that require network access.`,
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

func setupCapacityManagerCLIFlags(cmd *cobra.Command, OS *ServeOptions) {
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
	cmd.PersistentFlags().StringSliceVar(
		&OS.JobExecutionTimeoutClientIDBypassList, "job-execution-timeout-bypass-client-id", OS.JobExecutionTimeoutClientIDBypassList,
		`List of IDs of clients that are allowed to bypass the job execution timeout check`,
	)
}

func setupLibp2pCLIFlags(cmd *cobra.Command, OS *ServeOptions) {
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

func getPeers(OS *ServeOptions) []multiaddr.Multiaddr {
	var peersStrings []string
	if OS.PeerConnect == "none" {
		peersStrings = []string{}
	} else if OS.PeerConnect == "" {
		peersStrings = system.Envs[system.GetEnvironment()].BootstrapAddresses
	} else {
		peersStrings = strings.Split(OS.PeerConnect, ",")
	}

	peers := make([]multiaddr.Multiaddr, len(peersStrings))
	for i, peer := range peersStrings {
		peers[i], _ = multiaddr.NewMultiaddr(peer)
	}
	return peers
}

func getJobSelectionConfig(OS *ServeOptions) model.JobSelectionPolicy {
	// construct the job selection policy from the CLI args
	typedJobSelectionDataLocality := model.Anywhere

	if OS.JobSelectionDataLocality == "anywhere" {
		typedJobSelectionDataLocality = model.Anywhere
	}

	jobSelectionPolicy := model.JobSelectionPolicy{
		Locality:            typedJobSelectionDataLocality,
		RejectStatelessJobs: OS.JobSelectionDataRejectStateless,
		AcceptNetworkedJobs: OS.JobSelectionDataAcceptNetworked,
		ProbeHTTP:           OS.JobSelectionProbeHTTP,
		ProbeExec:           OS.JobSelectionProbeExec,
	}

	return jobSelectionPolicy
}

func getComputeConfig(OS *ServeOptions) node.ComputeConfig {
	return node.NewComputeConfigWith(node.ComputeConfigParams{
		JobSelectionPolicy: getJobSelectionConfig(OS),
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
		IgnorePhysicalResourceLimits:          os.Getenv("BACALHAU_CAPACITY_MANAGER_OVER_COMMIT") != "",
		JobExecutionTimeoutClientIDBypassList: OS.JobExecutionTimeoutClientIDBypassList,
	})
}

func newServeCmd() *cobra.Command {
	OS := NewServeOptions()

	serveCmd := &cobra.Command{
		Use:     "serve",
		Short:   "Start the bacalhau compute node",
		Long:    serveLong,
		Example: serveExample,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return serve(cmd, OS)
		},
	}

	serveCmd.PersistentFlags().StringSliceVar(
		&OS.NodeType, "node-type", OS.NodeType,
		`Whether the node is a compute, requester or both.`,
	)

	serveCmd.PersistentFlags().StringToStringVar(
		&OS.Labels, "labels", OS.Labels,
		`Labels to be associated with the node that can be used for node selection and filtering. (e.g. --labels key1=value1,key2=value2)`,
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

	setupLibp2pCLIFlags(serveCmd, OS)
	setupJobSelectionCLIFlags(serveCmd, OS)
	setupCapacityManagerCLIFlags(serveCmd, OS)

	return serveCmd
}

//nolint:funlen
func serve(cmd *cobra.Command, OS *ServeOptions) error {
	// Cleanup manager ensures that resources are freed before exiting:
	cm := system.NewCleanupManager()
	cm.RegisterCallback(system.CleanupTraceProvider)
	defer cm.Cleanup()

	// Context ensures main goroutine waits until killed with ctrl+c:
	ctx, cancel := signal.NotifyContext(cmd.Context(), ShutdownSignals...)
	defer cancel()

	ctx, rootSpan := system.NewRootSpan(ctx, system.GetTracer(), "cmd/bacalhau/serve")
	defer rootSpan.End()
	cm.RegisterCallback(system.CleanupTraceProvider)

	isComputeNode, isRequesterNode := false, false
	for _, nodeType := range OS.NodeType {
		if nodeType == "compute" {
			isComputeNode = true
		} else if nodeType == "requester" {
			isRequesterNode = true
		} else {
			return fmt.Errorf("invalid node type %s. Only compute and requester values are supported", nodeType)
		}
	}

	if OS.JobSelectionDataLocality != "local" && OS.JobSelectionDataLocality != "anywhere" {
		return fmt.Errorf("--job-selection-data-locality must be either 'local' or 'anywhere'")
	}

	// Establishing p2p connection
	peers := getPeers(OS)
	log.Debug().Msgf("libp2p connecting to: %s", peers)

	libp2pHost, err := libp2p.NewHost(OS.SwarmPort, rcmgr.DefaultResourceManager)
	if err != nil {
		Fatal(cmd, fmt.Sprintf("Error creating libp2p host: %s", err), 1)
	}
	cm.RegisterCallback(libp2pHost.Close)

	// add nodeID to logging context
	ctx = logger.ContextWithNodeIDLogger(ctx, libp2pHost.ID().String())

	// Establishing IPFS connection
	ipfsClient, err := ipfsClient(ctx, OS, cm)
	if err != nil {
		return err
	}

	datastore, err := inmemory.NewInMemoryDatastore()
	if err != nil {
		return fmt.Errorf("error creating in memory datastore: %s", err)
	}

	// Create node config from cmd arguments
	nodeConfig := node.NodeConfig{
		IPFSClient:           ipfsClient,
		CleanupManager:       cm,
		LocalDB:              datastore,
		Host:                 libp2pHost,
		FilecoinUnsealedPath: OS.FilecoinUnsealedPath,
		EstuaryAPIKey:        OS.EstuaryAPIKey,
		HostAddress:          OS.HostAddress,
		APIPort:              apiPort,
		MetricsPort:          OS.MetricsPort,
		ComputeConfig:        getComputeConfig(OS),
		RequesterNodeConfig:  node.NewRequesterConfigWithDefaults(),
		IsComputeNode:        isComputeNode,
		IsRequesterNode:      isRequesterNode,
		Labels:               OS.Labels,
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
	standardNode, err := node.NewStandardNode(ctx, nodeConfig)
	if err != nil {
		return fmt.Errorf("error creating node: %s", err)
	}

	// Start transport layer
	err = libp2p.ConnectToPeersContinuously(ctx, cm, libp2pHost, peers)
	if err != nil {
		return err
	}

	// Start node
	err = standardNode.Start(ctx)
	if err != nil {
		return fmt.Errorf("error starting node: %s", err)
	}

	<-ctx.Done() // block until killed
	return nil
}

func ipfsClient(ctx context.Context, OS *ServeOptions, cm *system.CleanupManager) (ipfs.Client, error) {
	if OS.IPFSConnect == "" {
		// Connect to the public IPFS nodes
		ipfsNode, err := ipfs.NewNode(ctx, cm, []string{})
		if err != nil {
			return ipfs.Client{}, fmt.Errorf("error creating IPFS node: %s", err)
		}
		client := ipfsNode.Client()

		swarmAddresses, err := client.SwarmAddresses(ctx)
		if err != nil {
			return ipfs.Client{}, fmt.Errorf("error looking up IPFS addresses: %s", err)
		}

		log.Ctx(ctx).Info().Strs("ipfs_swarm_addresses", swarmAddresses).Msg("Internal IPFS node available")
		return client, nil
	}

	client, err := ipfs.NewClientUsingRemoteHandler(OS.IPFSConnect)
	if err != nil {
		return ipfs.Client{}, fmt.Errorf("error creating IPFS client: %s", err)
	}

	return client, nil
}
