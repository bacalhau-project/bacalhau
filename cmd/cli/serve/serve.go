package serve

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"

	"github.com/libp2p/go-libp2p"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	computenodeapi "github.com/bacalhau-project/bacalhau/pkg/compute/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	bac_libp2p "github.com/bacalhau-project/bacalhau/pkg/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/libp2p/rcmgr"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/system/cleanup"
	"github.com/bacalhau-project/bacalhau/pkg/system/environment"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"

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

func GetServerOptions() (*ServeOptions, error) {
	engineStrs := viper.GetStringSlice(config.NodeDisabledFeaturesEngines)
	var engines []model.Engine
	for _, e := range engineStrs {
		engine, err := model.ParseEngine(e)
		if err != nil {
			return nil, err
		}
		engines = append(engines, engine)
	}
	publishersStrs := viper.GetStringSlice(config.NodeDisabledFeaturesPublishers)
	var publishers []model.Publisher
	for _, p := range publishersStrs {
		publisher, err := model.ParsePublisher(p)
		if err != nil {
			return nil, err
		}
		publishers = append(publishers, publisher)
	}
	storagesStrs := viper.GetStringSlice(config.NodeDisabledFeaturesStorages)
	var storages []model.StorageSourceType
	for _, s := range storagesStrs {
		storageType, err := model.ParseStorageSourceType(s)
		if err != nil {
			return nil, err
		}
		storages = append(storages, storageType)
	}
	AutoLabels := AutoOutputLabels()
	combinedLabelMap := make(map[string]string)
	for key, value := range AutoLabels {
		combinedLabelMap[key] = value
	}

	for key, value := range viper.GetStringMapString(config.NodeLabels) {
		combinedLabelMap[key] = value
	}

	jobLocality, err := model.ParseJobSelectionDataLocality(viper.GetString(config.NodeRequesterJobSelectionPolicyLocality))
	if err != nil {
		return nil, err
	}

	return &ServeOptions{
		NodeType:      viper.GetStringSlice(config.NodeType),
		PeerConnect:   viper.GetString(config.NodeLibp2pPeerConnect),
		IPFSConnect:   viper.GetString(config.NodeIPFSConnect),
		EstuaryAPIKey: viper.GetString(config.NodeEstuaryAPIKey),
		HostAddress:   "0.0.0.0", // TODO
		SwarmPort:     viper.GetInt(config.NodeLibp2pSwarmPort),
		JobSelectionPolicy: model.JobSelectionPolicy{
			Locality:            jobLocality,
			RejectStatelessJobs: viper.GetBool(config.NodeRequesterJobSelectionPolicyRejectStatelessJobs),
			AcceptNetworkedJobs: viper.GetBool(config.NodeRequesterJobSelectionPolicyAcceptNetworkedJobs),
			ProbeHTTP:           viper.GetString(config.NodeRequesterJobSelectionPolicyProbeHTTP),
			ProbeExec:           viper.GetString(config.NodeRequesterJobSelectionPolicyProbeExec),
		},
		ExternalVerifierHook: nil, //TODO currently there isn't a flag for this
		LimitTotalCPU:        viper.GetString(config.NodeComputeCapacityTotalCPU),
		LimitTotalMemory:     viper.GetString(config.NodeComputeCapacityTotalMemory),
		LimitTotalGPU:        viper.GetString(config.NodeComputeCapacityTotalGPU),
		LimitJobCPU:          viper.GetString(config.NodeComputeCapacityJobCPU),
		LimitJobMemory:       viper.GetString(config.NodeComputeCapacityJobMemory),
		LimitJobGPU:          viper.GetString(config.NodeComputeCapacityJobGPU),
		DisabledFeatures: node.FeatureConfig{
			Engines:    engines,
			Publishers: publishers,
			Storages:   storages,
		},
		JobExecutionTimeoutClientIDBypassList: viper.GetStringSlice(config.NodeComputeClientIDBypass),
		Labels:                                combinedLabelMap,
		IPFSSwarmAddresses:                    viper.GetStringSlice(config.NodeIPFSSwarmAddresses),
		PrivateInternalIPFS:                   viper.GetBool(config.NodeIPFSPrivateInternal),
		AllowListedLocalPaths:                 viper.GetStringSlice(config.NodeAllowListedLocalPaths),
	}, nil

}

//nolint:lll // Documentation
type ServeOptions struct {
	NodeType                              []string                 // "compute", "requester" node or both
	PeerConnect                           string                   // The libp2p multiaddress to connect to.
	IPFSConnect                           string                   // The multiaddress to connect to for IPFS.
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
	JobExecutionTimeoutClientIDBypassList []string                 // IDs of clients that can submit jobs more than the configured job execution timeout
	Labels                                map[string]string        // Labels to apply to the node that can be used for node selection and filtering
	IPFSSwarmAddresses                    []string                 // IPFS multiaddresses that the in-process IPFS should connect to
	PrivateInternalIPFS                   bool                     // Whether the in-process IPFS should automatically discover other IPFS nodes
	AllowListedLocalPaths                 []string                 // Local paths that are allowed to be mounted into jobs
}

func NewServeOptions() *ServeOptions {
	return &ServeOptions{
		NodeType:            []string{"requester"},
		PeerConnect:         DefaultPeerConnect,
		IPFSConnect:         "",
		EstuaryAPIKey:       os.Getenv("ESTUARY_API_KEY"),
		HostAddress:         "0.0.0.0",
		SwarmPort:           DefaultSwarmPort,
		JobSelectionPolicy:  model.NewDefaultJobSelectionPolicy(),
		LimitTotalCPU:       "",
		LimitTotalMemory:    "",
		LimitTotalGPU:       "",
		LimitJobCPU:         "",
		LimitJobMemory:      "",
		LimitJobGPU:         "",
		PrivateInternalIPFS: true,
	}
}

func SetupCapacityManagerCLIFlags(cmd *cobra.Command, OS *ServeOptions) {
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

func SetupLibp2pCLIFlags(cmd *cobra.Command, OS *ServeOptions) {
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

func GetPeers(OS *ServeOptions) ([]multiaddr.Multiaddr, error) {
	var peersStrings []string
	if OS.PeerConnect == DefaultPeerConnect {
		peersStrings = []string{}
	} else if OS.PeerConnect == "env" {
		peersStrings = system.Envs[environment.GetEnvironment()].BootstrapAddresses
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

func GetComputeConfig(OS *ServeOptions) node.ComputeConfig {
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

func GetRequesterConfig(OS *ServeOptions) node.RequesterConfig {
	return node.NewRequesterConfigWith(node.RequesterConfigParams{
		JobSelectionPolicy:       OS.JobSelectionPolicy,
		ExternalValidatorWebhook: OS.ExternalVerifierHook,
	})
}

func NewCmd() *cobra.Command {
	var options *ServeOptions
	serveCmd := &cobra.Command{
		Use:     "serve",
		Short:   "Start the bacalhau compute node",
		Long:    serveLong,
		Example: serveExample,
		Run: func(cmd *cobra.Command, _ []string) {
			var err error
			options, err = GetServerOptions()
			if err != nil {
				util.Fatal(cmd, err, 1)
			}
			if err := serve(cmd, options); err != nil {
				util.Fatal(cmd, err, 1)
			}
		},
	}
	if err := registerFlags(serveCmd, map[string][]flagDefinition{
		"libp2p":           Libp2pFlags,
		"ipfs":             IPFSFlags,
		"capacity":         CapacityFlags,
		"job-selection":    JobSelectionFlags,
		"disable-features": DisabledFeatureFlags,
		"labels":           LabelFlags,
		"node-type":        NodeTypeFlags,
		"estuary":          EstuaryFlags,
		"list-local":       AllowListLocalPathsFlags,
	}); err != nil {
		util.Fatal(serveCmd, err, 1)
	}

	return serveCmd
}

//nolint:funlen,gocyclo
func serve(cmd *cobra.Command, OS *ServeOptions) error {
	ctx := cmd.Context()
	cm := util.GetCleanupManager(ctx)

	fsRepo, err := repo.NewFS(viper.GetString("repo"))
	if err != nil {
		return err
	}
	if err := fsRepo.Init(); err != nil {
		return err
	}

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
	peers, err := GetPeers(OS)
	if err != nil {
		return err
	}
	log.Ctx(ctx).Debug().Msgf("libp2p connecting to: %s", peers)

	privKey, err := config.GetLibp2pPrivKey()
	if err != nil {
		return err
	}
	var libp2pOpts []libp2p.Option
	libp2pOpts = append(libp2pOpts, rcmgr.DefaultResourceManager, libp2p.Identity(privKey))
	libp2pHost, err := bac_libp2p.NewHost(OS.SwarmPort, libp2pOpts...)
	if err != nil {
		return fmt.Errorf("error creating libp2p host: %w", err)
	}
	cm.RegisterCallback(libp2pHost.Close)

	// add nodeID to logging context
	ctx = logger.ContextWithNodeIDLogger(ctx, libp2pHost.ID().String())

	// Establishing IPFS connection
	ipfsClient, err := IpfsClient(ctx, OS, cm)
	if err != nil {
		return err
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
		Host:                  libp2pHost,
		EstuaryAPIKey:         OS.EstuaryAPIKey,
		DisabledFeatures:      OS.DisabledFeatures,
		HostAddress:           OS.HostAddress,
		APIPort:               config.GetAPIPort(),
		ComputeConfig:         GetComputeConfig(OS),
		RequesterNodeConfig:   GetRequesterConfig(OS),
		IsComputeNode:         isComputeNode,
		IsRequesterNode:       isRequesterNode,
		Labels:                combinedMap,
		AllowListedLocalPaths: OS.AllowListedLocalPaths,
		FsRepo:                fsRepo,
	}

	// Create node
	standardNode, err := node.NewNode(ctx, nodeConfig)
	if err != nil {
		return fmt.Errorf("error creating node: %w", err)
	}

	// Start transport layer
	err = bac_libp2p.ConnectToPeersContinuously(ctx, cm, libp2pHost, peers)
	if err != nil {
		return err
	}

	// Start node
	if err := standardNode.Start(ctx); err != nil {
		return fmt.Errorf("error starting node: %w", err)
	}

	// only in station logging output
	if config.GetLogMode() == logger.LogModeStation && standardNode.IsComputeNode() {
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
			return fmt.Errorf("error looking up IPFS addresses: %w", err)
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

		summaryBuilder := strings.Builder{}
		summaryBuilder.WriteString(fmt.Sprintf(
			"export %s=%s\n",
			config.KeyAsEnvVar(config.NodeIPFSSwarmAddresses),
			ipfsSwarmAddress,
		))
		summaryBuilder.WriteString(fmt.Sprintf(
			"export %s=%s\n",
			config.KeyAsEnvVar(config.NodeAPIHost),
			OS.HostAddress,
		))
		summaryBuilder.WriteString(fmt.Sprintf(
			"export %s=%d\n",
			config.KeyAsEnvVar(config.NodeAPIPort),
			config.GetAPIPort(),
		))
		summaryBuilder.WriteString(fmt.Sprintf(
			"export %s=%s\n",
			config.KeyAsEnvVar(config.NodeLibp2pPeerConnect),
			peerAddress,
		))

		// Just convenience below - print out the last of the nodes information as the global variable
		summaryShellVariablesString := summaryBuilder.String()

		if isRequesterNode {
			cmd.Println()
			cmd.Println("To use this requester node from the client, run the following commands in your shell:")
			cmd.Println(summaryShellVariablesString)
		}

		ripath, err := fsRepo.WriteRunInfo(ctx, summaryShellVariablesString)
		if err != nil {
			return fmt.Errorf("writing run info to repo: %w", err)
		} else {
			cmd.Printf("A copy of these variables have been written to: %s\n", ripath)
		}
		if err != nil {
			return err
		}

		cm.RegisterCallback(func() error {
			return os.Remove(ripath)
		})

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

func IpfsClient(ctx context.Context, OS *ServeOptions, cm *cleanup.CleanupManager) (ipfs.Client, error) {
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
	// 	if !file.IsDir() && filepath.Ext(file.FlagName()) == ".list" {

	// 		packageList = append(packageList, file.FlagName()[:len(file.FlagName())-5])
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
