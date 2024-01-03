package serve

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/configflags"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	bac_libp2p "github.com/bacalhau-project/bacalhau/pkg/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
	"github.com/bacalhau-project/bacalhau/webui"

	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"
)

var DefaultSwarmPort = 1235

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

		# Start a public bacalhau node with the WebUI on port 3000 (default:80)
		bacalhau serve --web-ui --web-ui-port=3000
`))
)

func GetPeers(peerConnect string) ([]multiaddr.Multiaddr, error) {
	var (
		peersStrings []string
	)
	// TODO(forrest): [ux] this is a really confusing way to configure bootstrap peers.
	// The convenience is nice by passing a single 'env' value, and can be improved with sane defaults commented
	// out in the config. If a user wants to connect then can pass the --peer flag or uncomment the config values.
	if peerConnect == DefaultPeerConnect || peerConnect == "" {
		return nil, nil
	} else if peerConnect == "env" {
		// TODO(forrest): [ux/sanity] in the future default to the value in the config file and remove system environment
		peersStrings = system.Envs[system.GetEnvironment()].BootstrapAddresses
	} else if peerConnect == "config" {
		// TODO(forrest): [ux] if the user explicitly passes the peer flag with value `config` read the
		// boostrap peer list from their config file.
		return config.GetBootstrapPeers()
	} else {
		peersStrings = strings.Split(peerConnect, ",")
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

func NewCmd() *cobra.Command {
	serveFlags := map[string][]configflags.Definition{
		"requester-tls":    configflags.RequesterTLSFlags,
		"server-api":       configflags.ServerAPIFlags,
		"cluster":          configflags.ClusterFlags,
		"libp2p":           configflags.Libp2pFlags,
		"ipfs":             configflags.IPFSFlags,
		"capacity":         configflags.CapacityFlags,
		"job-timeouts":     configflags.ComputeTimeoutFlags,
		"job-selection":    configflags.JobSelectionFlags,
		"disable-features": configflags.DisabledFeatureFlags,
		"labels":           configflags.LabelFlags,
		"node-type":        configflags.NodeTypeFlags,
		"list-local":       configflags.AllowListLocalPathsFlags,
		"compute-store":    configflags.ComputeStorageFlags,
		"requester-store":  configflags.RequesterJobStorageFlags,
		"web-ui":           configflags.WebUIFlags,
		"node-info-store":  configflags.NodeInfoStoreFlags,
		"translations":     configflags.JobTranslationFlags,
	}

	serveCmd := &cobra.Command{
		Use:     "serve",
		Short:   "Start the bacalhau compute node",
		Long:    serveLong,
		Example: serveExample,
		PreRun: func(cmd *cobra.Command, args []string) {
			/*
				NB(forrest):
				(I learned a lot more about viper and cobra than was intended...)

				Binding flags in the PreRun phase is crucial to ensure that Viper binds only
				to the flags specific to the current command being executed. This helps prevent
				potential issues with overlapping flag names or default values from other commands.
				An example of an overlapping flagset is the libp2p flags, shared here and in the id command.
				By binding in PreRun, we maintain a clean separation between flag registration
				and its binding to configuration. It ensures that each command's flags are
				independently managed, avoiding interference or unexpected behavior from shared
				flag names across multiple commands.

				It's essential to understand the nature of Viper when working with Cobra commands.
				At its core, Viper functions as a flat namespace key-value store for configuration
				settings. It doesn't inherently respect or understand Cobra's command hierarchy or
				nuances. When multiple commands have overlapping flag names and modify the same
				configuration key in Viper, there's potential for confusion. For example, if two
				commands both use a "peer" flag, which value should Viper return and how does it
				know if the flag changes? Since Viper doesn't recognize the context of commands, it will
				return the value of the last flag bound to it. This is why it's important to manage
				flag binding thoughtfully, ensuring each command's context is respected.
			*/
			if err := configflags.BindFlags(cmd, serveFlags); err != nil {
				util.Fatal(cmd, err, 1)
			}
		},
		Run: func(cmd *cobra.Command, _ []string) {
			if err := serve(cmd); err != nil {
				util.Fatal(cmd, err, 1)
			}
		},
	}

	if err := configflags.RegisterFlags(serveCmd, serveFlags); err != nil {
		util.Fatal(serveCmd, err, 1)
	}

	return serveCmd
}

//nolint:funlen,gocyclo
func serve(cmd *cobra.Command) error {
	ctx := cmd.Context()
	cm := util.GetCleanupManager(ctx)

	// load the repo and its config file, reading in the values, flags and env vars will override values in config.
	repoDir, err := config.Get[string]("repo")
	if err != nil {
		return err
	}
	fsRepo, err := repo.NewFS(repoDir)
	if err != nil {
		return err
	}
	if err := fsRepo.Open(); err != nil {
		return err
	}

	// for now, use libp2p host ID as node ID, regardless of using NATS or Libp2p
	// TODO: allow users to specify node ID
	var nodeID string
	privKey, err := config.GetLibp2pPrivKey()
	if err != nil {
		return err
	}
	peerID, err := peer.IDFromPrivateKey(privKey)
	if err != nil {
		return err
	}
	nodeID = peerID.String()

	// configure node type
	isRequesterNode, isComputeNode, err := getNodeType()
	if err != nil {
		return err
	}

	// Establishing IPFS connection
	ipfsConfig, err := getIPFSConfig()
	if err != nil {
		return err
	}

	ipfsClient, err := SetupIPFSClient(ctx, cm, ipfsConfig)
	if err != nil {
		return err
	}

	clusterConfig, err := getClusterConfig()
	if err != nil {
		return err
	}

	libp2pCfg, err := config.GetLibp2pConfig()
	if err != nil {
		return err
	}

	if !clusterConfig.UseNATS {
		peers, err := GetPeers(libp2pCfg.PeerConnect)
		if err != nil {
			return err
		}
		// configure libp2p
		libp2pHost, err := setupLibp2pHost(libp2pCfg, privKey)
		if err != nil {
			return err
		}
		cm.RegisterCallback(libp2pHost.Close)

		// Start transport layer
		err = bac_libp2p.ConnectToPeersContinuously(ctx, cm, libp2pHost, peers)
		if err != nil {
			return err
		}

		clusterConfig.Libp2pHost = libp2pHost
		nodeID = libp2pHost.ID().String()
	}

	// add nodeID to logging context
	ctx = logger.ContextWithNodeIDLogger(ctx, nodeID)

	computeConfig, err := GetComputeConfig()
	if err != nil {
		return err
	}

	requesterConfig, err := GetRequesterConfig()
	if err != nil {
		return err
	}

	featureConfig, err := getDisabledFeatures()
	if err != nil {
		return err
	}

	nodeInfoStoreTTL, err := config.Get[time.Duration](types.NodeNodeInfoStoreTTL)
	if err != nil {
		return err
	}

	allowedListLocalPaths := getAllowListedLocalPathsConfig()

	// Create node config from cmd arguments
	nodeConfig := node.NodeConfig{
		NodeID:                nodeID,
		CleanupManager:        cm,
		IPFSClient:            ipfsClient,
		DisabledFeatures:      featureConfig,
		HostAddress:           config.ServerAPIHost(),
		APIPort:               config.ServerAPIPort(),
		ComputeConfig:         computeConfig,
		RequesterNodeConfig:   requesterConfig,
		IsComputeNode:         isComputeNode,
		IsRequesterNode:       isRequesterNode,
		Labels:                config.GetStringMapString(types.NodeLabels),
		AllowListedLocalPaths: allowedListLocalPaths,
		FsRepo:                fsRepo,
		NodeInfoStoreTTL:      nodeInfoStoreTTL,
		ClusterConfig:         clusterConfig,
	}

	if isRequesterNode {
		// We only want auto TLS for the requester node, but this info doesn't fit well
		// with the other data in the requesterConfig.
		nodeConfig.RequesterAutoCert = config.ServerAutoCertDomain()
		nodeConfig.RequesterAutoCertCache = config.GetAutoCertCachePath()

		cert, key := config.GetRequesterCertificateSettings()
		nodeConfig.RequesterTLSCertificateFile = cert
		nodeConfig.RequesterTLSKeyFile = key
	}

	// Create node
	standardNode, err := node.NewNode(ctx, nodeConfig)
	if err != nil {
		return fmt.Errorf("error creating node: %w", err)
	}

	// Start node
	if err := standardNode.Start(ctx); err != nil {
		return fmt.Errorf("error starting node: %w", err)
	}

	startWebUI, err := config.Get[bool](types.NodeWebUIEnabled)
	if err != nil {
		return err
	}

	// Start up Dashboard
	if startWebUI {
		listenPort, err := config.Get[int](types.NodeWebUIPort)
		if err != nil {
			return err
		}

		apiURL := standardNode.APIServer.GetURI().JoinPath("api", "v1")
		go func() {
			// Specifically leave the host blank. The app will just use whatever
			// host it is served on and replace the port and path.
			apiPort := apiURL.Port()
			apiPath := apiURL.Path

			err := webui.ListenAndServe(ctx, "", apiPort, apiPath, listenPort)
			if err != nil {
				cmd.PrintErrln(err)
			}
		}()
	}

	// only in station logging output
	if config.GetLogMode() == logger.LogModeStation && standardNode.IsComputeNode() {
		cmd.Printf("API: %s\n", standardNode.APIServer.GetURI().JoinPath("/api/v1/compute/debug"))
	}

	// Print out node info and shell variables to connect to this node
	envVarBuilder := strings.Builder{}
	envVarBuilder.WriteString(fmt.Sprintf(
		"export %s=%s\n",
		config.KeyAsEnvVar(types.NodeClientAPIHost),
		config.ServerAPIHost(),
	))
	envVarBuilder.WriteString(fmt.Sprintf(
		"export %s=%d\n",
		config.KeyAsEnvVar(types.NodeClientAPIPort),
		config.ServerAPIPort(),
	))

	sb := strings.Builder{}
	if isRequesterNode {
		sb.WriteString("To connect another node to this one, run the following command in your shell:\n")

		sb.WriteString(fmt.Sprintf("%s serve ", os.Args[0]))
		// other nodes can be just compute nodes
		// no need to spawn 1+ requester nodes
		sb.WriteString(fmt.Sprintf("%s=compute ",
			configflags.FlagNameForKey(types.NodeType, configflags.NodeTypeFlags...)))

		if clusterConfig.UseNATS {
			sb.WriteString(fmt.Sprintf("%s ",
				configflags.FlagNameForKey(types.NodeClusterUseNATS, configflags.ClusterFlags...)))
			sb.WriteString(fmt.Sprintf("%s=%s:%d ",
				configflags.FlagNameForKey(types.NodeClusterOrchestrators, configflags.ClusterFlags...),
				clusterConfig.AdvertisedAddress, clusterConfig.Port,
			))
			envVarBuilder.WriteString(fmt.Sprintf(
				"export %s=%v\n",
				config.KeyAsEnvVar(types.NodeClusterUseNATS),
				true,
			))
			envVarBuilder.WriteString(fmt.Sprintf(
				"export %s=%s:%d\n",
				config.KeyAsEnvVar(types.NodeClusterOrchestrators),
				clusterConfig.AdvertisedAddress, clusterConfig.Port,
			))
		} else {
			p2pAddr, err := multiaddr.NewMultiaddr("/p2p/" + clusterConfig.Libp2pHost.ID().String())
			if err != nil {
				return err
			}
			peerAddress := pickP2pAddress(clusterConfig.Libp2pHost.Addrs()).Encapsulate(p2pAddr).String()
			sb.WriteString(fmt.Sprintf("%s=%s ",
				configflags.FlagNameForKey(types.NodeLibp2pPeerConnect, configflags.Libp2pFlags...),
				peerAddress,
			))
			envVarBuilder.WriteString(fmt.Sprintf(
				"export %s=%s\n",
				config.KeyAsEnvVar(types.NodeLibp2pPeerConnect),
				peerAddress,
			))
		}
	} else {
		if !clusterConfig.UseNATS {
			cmd.Println("Make sure there's at least one requester node in your network.")
		}
	}

	if ipfsConfig.PrivateInternal {
		ipfsAddresses, err := ipfsClient.SwarmMultiAddresses(ctx)
		if err != nil {
			return fmt.Errorf("error looking up IPFS addresses: %w", err)
		}

		ipfsSwarmAddress := pickP2pAddress(ipfsAddresses).String()
		sb.WriteString(fmt.Sprintf("%s ",
			configflags.FlagNameForKey(types.NodeIPFSPrivateInternal, configflags.IPFSFlags...)))

		sb.WriteString(fmt.Sprintf("%s=%s ",
			configflags.FlagNameForKey(types.NodeIPFSSwarmAddresses, configflags.IPFSFlags...),
			ipfsSwarmAddress,
		))

		envVarBuilder.WriteString(fmt.Sprintf(
			"export %s=%s\n",
			config.KeyAsEnvVar(types.NodeIPFSSwarmAddresses),
			ipfsSwarmAddress,
		))
	}

	cmd.Println()
	cmd.Println(sb.String())

	summaryShellVariablesString := envVarBuilder.String()
	cmd.Println()
	cmd.Println("To connect to this node from the client, run the following commands in your shell:")
	cmd.Println(summaryShellVariablesString)

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
