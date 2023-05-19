package bacalhau

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	computenodeapi "github.com/bacalhau-project/bacalhau/pkg/compute/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore/inmemory"
	"github.com/bacalhau-project/bacalhau/pkg/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/libp2p/rcmgr"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	filecoinlotus "github.com/bacalhau-project/bacalhau/pkg/publisher/filecoin_lotus"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
	"github.com/multiformats/go-multiaddr"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"
)

var DefaultSwarmPort = 1235

const NvidiaCLI = "nvidia-container-cli"
const DefaultPeerConnect = "none"

var (
	serveLong = templates.LongDesc(i18n.T(`
		Start a bacalhau node.
		`))

	serveExample = templates.Examples(i18n.T(`
		# Start a private bacalhau requester node
		bacalhau serve
		# or
		bacalhau serve --node-type requester

		# Start a private bacalhau hybrid node that acts as both compute and requester
		bacalhau serve --node-type compute --node-type requester
		# or
		bacalhau serve --node-type compute,requester

		# Start a private bacalhau node with a persistent local IPFS node
		BACALHAU_SERVE_IPFS_PATH=/data/ipfs bacalhau serve

		# Start a public bacalhau requester node
		bacalhau serve --peer env --private-internal-ipfs=false
`))
)

//nolint:lll // Documentation
type ServeOptions struct {
	NodeType                              []string                 // "compute", "requester" node or both
	PeerConnect                           string                   // The libp2p multiaddress to connect to.
	IPFSConnect                           string                   // The multiaddress to connect to for IPFS.
	FilecoinUnsealedPath                  string                   // Go template to turn a Filecoin CID into a local filepath with the unsealed data.
	EstuaryAPIKey                         string                   // The API key used when using the estuary API.
	HostAddress                           string                   // The host address to listen on.
	SwarmPort                             int                      // The host port for libp2p network.
	JobSelectionPolicy                    model.JobSelectionPolicy // How the node decides what jobs to run.
	ExternalVerifierHook                  *url.URL                 // Where to send external verification requests to.
	LimitTotalCPU                         string                   // The total amount of CPU the system can be using at one time.
	LimitTotalMemory                      string                   // The total amount of memory the system can be using at one time.
	LimitTotalGPU                         string                   // The total amount of GPU the system can be using at one time.
	LimitJobCPU                           string                   // The amount of CPU the system can be using at one time for a single job.
	LimitJobMemory                        string                   // The amount of memory the system can be using at one time for a single job.
	LimitJobGPU                           string                   // The amount of GPU the system can be using at one time for a single job.
	DisabledFeatures                      node.FeatureConfig       // What feautres should not be enbaled even if installed
	LotusFilecoinStorageDuration          time.Duration            // How long deals should be for the Lotus Filecoin publisher
	LotusFilecoinPathDirectory            string                   // The location of the Lotus configuration directory which contains config.toml, etc
	LotusFilecoinUploadDirectory          string                   // Directory to put files when uploading to Lotus (optional)
	LotusFilecoinMaximumPing              time.Duration            // The maximum ping allowed when selecting a Filecoin miner
	JobExecutionTimeoutClientIDBypassList []string                 // IDs of clients that can submit jobs more than the configured job execution timeout
	Labels                                map[string]string        // Labels to apply to the node that can be used for node selection and filtering
	IPFSSwarmAddresses                    []string                 // IPFS multiaddresses that the in-process IPFS should connect to
	PrivateInternalIPFS                   bool                     // Whether the in-process IPFS should automatically discover other IPFS nodes
	AllowListedLocalPaths                 []string                 // Local paths that are allowed to be mounted into jobs
}

func NewServeOptions() *ServeOptions {
	return &ServeOptions{
		NodeType:                   []string{"requester"},
		PeerConnect:                DefaultPeerConnect,
		IPFSConnect:                "",
		FilecoinUnsealedPath:       "",
		EstuaryAPIKey:              os.Getenv("ESTUARY_API_KEY"),
		HostAddress:                "0.0.0.0",
		SwarmPort:                  DefaultSwarmPort,
		JobSelectionPolicy:         model.NewDefaultJobSelectionPolicy(),
		LimitTotalCPU:              "",
		LimitTotalMemory:           "",
		LimitTotalGPU:              "",
		LimitJobCPU:                "",
		LimitJobMemory:             "",
		LimitJobGPU:                "",
		LotusFilecoinPathDirectory: os.Getenv("LOTUS_PATH"),
		LotusFilecoinMaximumPing:   2 * time.Second,
		PrivateInternalIPFS:        true,
	}
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
		`A comma-separated list of libp2p multiaddress to connect to. `+
			`Use "none" to avoid connecting to any peer, `+
			`"env" to connect to the default peer list of your active environment (see BACALHAU_ENVIRONMENT env var).`,
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

func getPeers(OS *ServeOptions) ([]multiaddr.Multiaddr, error) {
	var peersStrings []string
	if OS.PeerConnect == DefaultPeerConnect {
		peersStrings = []string{}
	} else if OS.PeerConnect == "env" {
		peersStrings = system.Envs[system.GetEnvironment()].BootstrapAddresses
	} else {
		peersStrings = strings.Split(OS.PeerConnect, ",")
	}

	peers := make([]multiaddr.Multiaddr, 0, len(peersStrings))
	for _, peer := range peersStrings {
		parsed, err := multiaddr.NewMultiaddr(peer)
		if err != nil {
			return nil, err
		}
		peers = append(peers, parsed)
	}
	return peers, nil
}

func getComputeConfig(OS *ServeOptions) node.ComputeConfig {
	return node.NewComputeConfigWith(node.ComputeConfigParams{
		JobSelectionPolicy: OS.JobSelectionPolicy,
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

func getRequesterConfig(OS *ServeOptions) node.RequesterConfig {
	return node.NewRequesterConfigWith(node.RequesterConfigParams{
		JobSelectionPolicy:       OS.JobSelectionPolicy,
		ExternalValidatorWebhook: OS.ExternalVerifierHook,
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
		`The ipfs host multiaddress to connect to, otherwise an in-process IPFS node will be created if not set.`,
	)
	serveCmd.PersistentFlags().StringVar(
		&OS.FilecoinUnsealedPath, "filecoin-unsealed-path", OS.FilecoinUnsealedPath,
		`The go template that can turn a filecoin CID into a local filepath with the unsealed data.`,
	)
	serveCmd.PersistentFlags().StringVar(
		&OS.EstuaryAPIKey, "estuary-api-key", OS.EstuaryAPIKey,
		`The API key used when using the estuary API.`,
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
	serveCmd.PersistentFlags().StringSliceVar(
		&OS.IPFSSwarmAddresses, "ipfs-swarm-addr", OS.IPFSSwarmAddresses,
		"IPFS multiaddress to connect the in-process IPFS node to - cannot be used with --ipfs-connect.",
	)
	serveCmd.PersistentFlags().StringSliceVar(
		&OS.AllowListedLocalPaths, "allow-listed-local-paths", OS.AllowListedLocalPaths,
		"Local paths that are allowed to be mounted into jobs",
	)
	serveCmd.PersistentFlags().Var(
		URLFlag(&OS.ExternalVerifierHook, "http"), "external-verifier-http",
		"An HTTP URL to which the verification request should be posted for jobs using the 'external' verifier. "+
			"The 'external' verifier will not be enabled if this is unset.",
	)
	serveCmd.PersistentFlags().BoolVar(
		&OS.PrivateInternalIPFS, "private-internal-ipfs", OS.PrivateInternalIPFS,
		"Whether the in-process IPFS node should auto-discover other nodes, including the public IPFS network - "+
			"cannot be used with --ipfs-connect. "+
			"Use \"--private-internal-ipfs=false\" to disable. "+
			"To persist a local Ipfs node, set BACALHAU_SERVE_IPFS_PATH to a valid path.",
	)

	setupLibp2pCLIFlags(serveCmd, OS)
	serveCmd.Flags().AddFlagSet(DisabledFeatureCLIFlags(&OS.DisabledFeatures))
	serveCmd.Flags().AddFlagSet(JobSelectionCLIFlags(&OS.JobSelectionPolicy))
	setupCapacityManagerCLIFlags(serveCmd, OS)

	return serveCmd
}

//nolint:funlen,gocyclo
func serve(cmd *cobra.Command, OS *ServeOptions) error {
	ctx := cmd.Context()
	cm := ctx.Value(systemManagerKey).(*system.CleanupManager)

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

	if OS.IPFSConnect != "" && OS.PrivateInternalIPFS {
		return fmt.Errorf("--private-internal-ipfs cannot be used with --ipfs-connect")
	}

	if OS.IPFSConnect != "" && len(OS.IPFSSwarmAddresses) != 0 {
		return fmt.Errorf("--ipfs-swarm-addr cannot be used with --ipfs-connect")
	}

	// Establishing p2p connection
	peers, err := getPeers(OS)
	if err != nil {
		return err
	}
	log.Ctx(ctx).Debug().Msgf("libp2p connecting to: %s", peers)

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

	datastore := inmemory.NewJobStore()
	if err != nil {
		return fmt.Errorf("error creating in memory datastore: %s", err)
	}
	AutoLabels := AutoOutputLabels()
	combinedMap := make(map[string]string)
	for key, value := range AutoLabels {
		combinedMap[key] = value
	}

	for key, value := range OS.Labels {
		combinedMap[key] = value
	}
	// Create node config from cmd arguments
	nodeConfig := node.NodeConfig{
		IPFSClient:            ipfsClient,
		CleanupManager:        cm,
		JobStore:              datastore,
		Host:                  libp2pHost,
		FilecoinUnsealedPath:  OS.FilecoinUnsealedPath,
		EstuaryAPIKey:         OS.EstuaryAPIKey,
		DisabledFeatures:      OS.DisabledFeatures,
		HostAddress:           OS.HostAddress,
		APIPort:               apiPort,
		ComputeConfig:         getComputeConfig(OS),
		RequesterNodeConfig:   getRequesterConfig(OS),
		IsComputeNode:         isComputeNode,
		IsRequesterNode:       isRequesterNode,
		Labels:                combinedMap,
		AllowListedLocalPaths: OS.AllowListedLocalPaths,
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
	standardNode, err := node.NewNode(ctx, nodeConfig)
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

	// only in station logging output
	if loggingMode == logger.LogModeStation && standardNode.IsComputeNode() {
		cmd.Printf("API: %s\n", standardNode.APIServer.GetURI().JoinPath(computenodeapi.APIPrefix, computenodeapi.APIDebugSuffix))
	}

	if OS.PrivateInternalIPFS && OS.PeerConnect == DefaultPeerConnect {
		// other nodes can be just compute nodes
		// no need to spawn 1+ requester nodes
		nodeType := "--node-type compute"

		if isComputeNode && !isRequesterNode {
			cmd.Println("Make sure there's at least one requester node in your network.")
		}

		ipfsAddresses, err := ipfsClient.SwarmMultiAddresses(ctx)
		if err != nil {
			return fmt.Errorf("error looking up IPFS addresses: %s", err)
		}

		p2pAddr, err := multiaddr.NewMultiaddr("/p2p/" + libp2pHost.ID().String())
		if err != nil {
			return err
		}

		peerAddress := pickP2pAddress(libp2pHost.Addrs()).Encapsulate(p2pAddr).String()
		ipfsSwarmAddress := pickP2pAddress(ipfsAddresses).String()

		cmd.Println()
		cmd.Println("To connect another node to this private one, run the following command in your shell:")
		cmd.Printf(
			"%s serve %s --private-internal-ipfs --peer %s --ipfs-swarm-addr %s\n",
			os.Args[0], nodeType, peerAddress, ipfsSwarmAddress,
		)

		if isRequesterNode {
			cmd.Println()
			cmd.Println("To use this requester node from the client, run the following commands in your shell:")
			cmd.Printf("export BACALHAU_IPFS_SWARM_ADDRESSES=%s\n", ipfsSwarmAddress)
			cmd.Printf("export BACALHAU_API_HOST=%s\n", OS.HostAddress)
			cmd.Printf("export BACALHAU_API_PORT=%d\n", apiPort)
		}
	}

	<-ctx.Done() // block until killed
	return nil
}

// pickP2pAddress will aim to select a non-localhost IPv4 TCP address, or at least a non-localhost IPv6 one, from a list
// of addresses.
func pickP2pAddress(addresses []multiaddr.Multiaddr) multiaddr.Multiaddr {
	value := func(m multiaddr.Multiaddr) int {
		count := 0
		if _, err := m.ValueForProtocol(multiaddr.P_TCP); err == nil {
			count++
		}
		if ip, err := m.ValueForProtocol(multiaddr.P_IP4); err == nil {
			count++
			if ip != "127.0.0.1" {
				count++
			}
		} else if ip, err := m.ValueForProtocol(multiaddr.P_IP6); err == nil && ip != "::1" {
			count++
		}
		return count
	}

	preferredAddress := config.PreferredAddress()
	if preferredAddress != "" {
		for _, addr := range addresses {
			if strings.Contains(addr.String(), preferredAddress) {
				return addr
			}
		}
	}

	sort.Slice(addresses, func(i, j int) bool {
		return value(addresses[i]) > value(addresses[j])
	})

	return addresses[0]
}

func ipfsClient(ctx context.Context, OS *ServeOptions, cm *system.CleanupManager) (ipfs.Client, error) {
	if OS.IPFSConnect == "" {
		// Connect to the public IPFS nodes by default
		newNode := ipfs.NewNode
		if OS.PrivateInternalIPFS {
			newNode = ipfs.NewLocalNode
		}

		ipfsNode, err := newNode(ctx, cm, OS.IPFSSwarmAddresses)
		if err != nil {
			return ipfs.Client{}, fmt.Errorf("error creating IPFS node: %s", err)
		}
		if OS.PrivateInternalIPFS {
			log.Ctx(ctx).Debug().Msgf("ipfs_node_apiport: %d", ipfsNode.APIPort)
		}
		cm.RegisterCallbackWithContext(ipfsNode.Close)
		client := ipfsNode.Client()

		swarmAddresses, err := client.SwarmAddresses(ctx)
		if err != nil {
			return ipfs.Client{}, fmt.Errorf("error looking up IPFS addresses: %s", err)
		}

		log.Ctx(ctx).Debug().Strs("ipfs_swarm_addresses", swarmAddresses).Msg("Internal IPFS node available")
		return client, nil
	}

	client, err := ipfs.NewClientUsingRemoteHandler(ctx, OS.IPFSConnect)
	if err != nil {
		return ipfs.Client{}, fmt.Errorf("error creating IPFS client: %s", err)
	}

	return client, nil
}
func AutoOutputLabels() map[string]string {
	m := make(map[string]string)
	// Get the operating system name
	os := runtime.GOOS
	m["Operating-System"] = os
	m["git-lfs"] = "False"
	if checkGitLFS() {
		m["git-lfs"] = "True"
	}
	arch := runtime.GOARCH
	m["Architecture"] = arch
	CLIPATH, _ := exec.LookPath(NvidiaCLI)
	if CLIPATH != "" {
		gpuNames, gpuMemory := gpuList()
		// Print the GPU names
		for i, name := range gpuNames {
			name = strings.Replace(name, " ", "-", -1) // Replace spaces with dashes
			key := fmt.Sprintf("GPU-%d", i)
			m[key] = name
			key = fmt.Sprintf("GPU-%d-Memory", i)
			memory := strings.Replace(gpuMemory[i], " ", "-", -1) // Replace spaces with dashes
			m[key] = memory
		}
	}
	// Get list of installed packages (Only works for linux, make it work for every platform)
	// files, err := ioutil.ReadDir("/var/lib/dpkg/info")
	// if err != nil {
	// 	panic(err)
	// }
	// var packageList []string
	// for _, file := range files {
	// 	if !file.IsDir() && filepath.Ext(file.Name()) == ".list" {

	// 		packageList = append(packageList, file.Name()[:len(file.Name())-5])
	// 	}
	// }
	// m["Installed-Packages"] = strings.Join(packageList, ",")
	return m
}

func gpuList() ([]string, []string) {
	// Execute nvidia-smi command to get GPU names
	cmd := exec.Command("nvidia-smi", "--query-gpu=gpu_name", "--format=csv")
	output, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	// Split the output by newline character
	gpuNames := strings.Split(string(output), "\n")

	// Remove the first and last elements of the slice
	gpuNames = gpuNames[1 : len(gpuNames)-1]

	cmd1 := exec.Command("nvidia-smi", "--query-gpu=memory.total", "--format=csv")
	output1, err1 := cmd1.Output()
	if err1 != nil {
		panic(err1)
	}

	// Split the output by newline character
	gpuMemory := strings.Split(string(output1), "\n")

	// Remove the first and last elements of the slice
	gpuMemory = gpuMemory[1 : len(gpuMemory)-1]

	return gpuNames, gpuMemory
}

func checkGitLFS() bool {
	_, err := exec.LookPath("git-lfs")
	return err == nil
}
