package serve

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/configflags"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/lib/crypto"
	bac_libp2p "github.com/bacalhau-project/bacalhau/pkg/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/libp2p/rcmgr"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
	"github.com/bacalhau-project/bacalhau/webui"
)

var DefaultSwarmPort = 1235
var DefaultWebPort = 8483

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

		# Start a public bacalhau node with the WebUI on port 3000 (default:8483)
		bacalhau serve --web-ui --web-ui-port=3000
`))
)

func GetPeers(cfg types.BacalhauConfig) ([]multiaddr.Multiaddr, error) {
	var (
		peersStrings []string
	)
	// TODO(forrest): [ux] this is a really confusing way to configure bootstrap peers.
	// The convenience is nice by passing a single 'env' value, and can be improved with sane defaults commented
	// out in the config. If a user wants to connect then can pass the --peer flag or uncomment the config values.
	if cfg.Node.Libp2p.PeerConnect == DefaultPeerConnect || cfg.Node.Libp2p.PeerConnect == "" {
		return nil, nil
	} else if cfg.Node.Libp2p.PeerConnect == "env" {
		// TODO(forrest): [ux/sanity] in the future default to the value in the config file and remove system environment
		peersStrings = system.Envs[system.GetEnvironment()].BootstrapAddresses
	} else if cfg.Node.Libp2p.PeerConnect == "config" {
		// TODO(forrest): [ux] if the user explicitly passes the peer flag with value `config` read the
		// bootstrap peer list from their config file.
		return parseBootstrapPeers(cfg.Node.BootstrapAddresses)
	} else {
		peersStrings = strings.Split(cfg.Node.Libp2p.PeerConnect, ",")
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
		"local_publisher":       configflags.LocalPublisherFlags,
		"publishing":            configflags.PublishingFlags,
		"requester-tls":         configflags.RequesterTLSFlags,
		"server-api":            configflags.ServerAPIFlags,
		"network":               configflags.NetworkFlags,
		"libp2p":                configflags.Libp2pFlags,
		"ipfs":                  configflags.IPFSFlags,
		"capacity":              configflags.CapacityFlags,
		"job-timeouts":          configflags.ComputeTimeoutFlags,
		"job-selection":         configflags.JobSelectionFlags,
		"disable-features":      configflags.DisabledFeatureFlags,
		"labels":                configflags.LabelFlags,
		"node-type":             configflags.NodeTypeFlags,
		"list-local":            configflags.AllowListLocalPathsFlags,
		"compute-store":         configflags.ComputeStorageFlags,
		"requester-store":       configflags.RequesterJobStorageFlags,
		"web-ui":                configflags.WebUIFlags,
		"node-info-store":       configflags.NodeInfoStoreFlags,
		"node-name":             configflags.NodeNameFlags,
		"translations":          configflags.JobTranslationFlags,
		"docker-cache-manifest": configflags.DockerManifestCacheFlags,
	}
	serveCmd := &cobra.Command{
		Use:     "serve",
		Short:   "Start the bacalhau compute node",
		Long:    serveLong,
		Example: serveExample,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return configflags.BindFlags(cmd, viper.GetViper(), serveFlags)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			// get the global viper instance
			v := viper.GetViper()
			// get the repo path set in the root command.
			repoPath := v.GetString("repo")
			if repoPath == "" {
				return fmt.Errorf("repo path not set")
			}
			cfg := config.New(config.WithViper(v))
			// create or open the bacalhau repo and load the config
			fsr, err := setup.SetupBacalhauRepo(repoPath, cfg)
			if err != nil {
				return fmt.Errorf("failed to reconcile repo: %w", err)
			}
			bcfg, err := cfg.Current()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			return serve(cmd, bcfg, fsr)
		},
	}

	if err := configflags.RegisterFlags(serveCmd, serveFlags); err != nil {
		util.Fatal(serveCmd, err, 1)
	}
	return serveCmd
}

//nolint:funlen,gocyclo
func serve(cmd *cobra.Command, cfg types.BacalhauConfig, fsRepo *repo.FsRepo) error {
	ctx := cmd.Context()
	cm := util.GetCleanupManager(ctx)

	var nodeName string
	var libp2pHost host.Host
	var libp2pPeers []string
	var err error

	// if the transport type is libp2p, we use the peerID as the node name
	// even if the user provided one to avoid issues with peer lookups
	if cfg.Node.Network.Type == models.NetworkTypeLibp2p {
		libp2pHost, libp2pPeers, err = setupLibp2p(cfg)
		if err != nil {
			return err
		}
		nodeName = libp2pHost.ID().String()
		cfg.Node.Name = nodeName
	} else {
		if cfg.Node.Name == "" {
			nodeName, err = getNodeID(ctx, cfg.Node.NameProvider)
			if err != nil {
				return err
			}
			cfg.Node.Name = nodeName
		}
	}
	ctx = logger.ContextWithNodeIDLogger(ctx, nodeName)

	// configure node type
	isRequesterNode, isComputeNode, err := getNodeType(cfg)
	if err != nil {
		return err
	}

	// Establishing IPFS connection
	ipfsConfig, err := getIPFSConfig(cfg.Node.IPFS)
	if err != nil {
		return err
	}

	ipfsClient, err := SetupIPFSClient(ctx, cm, ipfsConfig)
	if err != nil {
		return err
	}

	networkConfig, err := getNetworkConfig(cfg.Node.Network)
	if err != nil {
		return err
	}

	if networkConfig.Type == models.NetworkTypeLibp2p {
		networkConfig.Libp2pHost = libp2pHost
		networkConfig.ClusterPeers = libp2pPeers
	}

	computeConfig, err := GetComputeConfig(ctx, cfg.Node, isComputeNode)
	if err != nil {
		return errors.Wrapf(err, "failed to configure compute node")
	}

	requesterConfig, err := GetRequesterConfig(ctx, cfg.Node.Requester, isRequesterNode)
	if err != nil {
		return errors.Wrapf(err, "failed to configure requester node")
	}

	// Create node config from cmd arguments
	hostAddress, err := parseServerAPIHost(cfg.Node.ServerAPI.Host)
	if err != nil {
		return err
	}
	nodeConfig := node.NodeConfig{
		NodeID:                cfg.Node.Name,
		CleanupManager:        cm,
		IPFSClient:            ipfsClient,
		DisabledFeatures:      node.FeatureConfig(cfg.Node.DisabledFeatures),
		HostAddress:           hostAddress,
		APIPort:               uint16(cfg.Node.ServerAPI.Port),
		ComputeConfig:         computeConfig,
		RequesterNodeConfig:   requesterConfig,
		AuthConfig:            cfg.Auth,
		IsComputeNode:         isComputeNode,
		IsRequesterNode:       isRequesterNode,
		RequesterSelfSign:     cfg.Node.ServerAPI.TLS.SelfSigned,
		Labels:                cfg.Node.Labels,
		AllowListedLocalPaths: cfg.Node.AllowListedLocalPaths,
		NodeInfoStoreTTL:      time.Duration(cfg.Node.NodeInfoStoreTTL),
		NetworkConfig:         networkConfig,
	}
	if isRequesterNode {
		// We only want auto TLS for the requester node, but this info doesn't fit well
		// with the other data in the requesterConfig.
		nodeConfig.RequesterAutoCert = cfg.Node.ServerAPI.TLS.AutoCert
		nodeConfig.RequesterAutoCertCache = cfg.Node.ServerAPI.TLS.AutoCertCachePath
		// If there are configuration values for autocert we should return and let autocert
		// do what it does later on in the setup.
		if nodeConfig.RequesterAutoCert == "" {
			cert, key, err := GetTLSCertificate(ctx, cfg, &nodeConfig)
			if err != nil {
				return err
			}
			nodeConfig.RequesterTLSCertificateFile = cert
			nodeConfig.RequesterTLSKeyFile = key
		}
	}
	// Create node
	standardNode, err := node.NewNode(ctx, cfg, nodeConfig, fsRepo)
	if err != nil {
		return fmt.Errorf("error creating node: %w", err)
	}

	// Persist the node config after the node is created and its config is valid.
	repoDir, err := fsRepo.Path()
	if err != nil {
		return err
	}
	if err = persistConfigs(repoDir, cfg); err != nil {
		return fmt.Errorf("error persisting configs: %w", err)
	}

	// Start node
	if err := standardNode.Start(ctx); err != nil {
		return fmt.Errorf("error starting node: %w", err)
	}

	// Start up Dashboard - default: 8483
	if cfg.Node.WebUI.Enabled {
		apiURL := standardNode.APIServer.GetURI().JoinPath("api", "v1")
		go func() {
			// Specifically leave the host blank. The app will just use whatever
			// host it is served on and replace the port and path.
			apiPort := apiURL.Port()
			apiPath := apiURL.Path

			err := webui.ListenAndServe(ctx, "", apiPort, apiPath, cfg.Node.WebUI.Port)
			if err != nil {
				cmd.PrintErrln(err)
			}
		}()
	}

	connectCmd, err := buildConnectCommand(ctx, &nodeConfig, ipfsConfig)
	if err != nil {
		return err
	}
	cmd.Println()
	cmd.Println(connectCmd)

	envVars, err := buildEnvVariables(ctx, cfg, &nodeConfig, ipfsConfig)
	if err != nil {
		return err
	}
	cmd.Println()
	cmd.Println("To connect to this node from the client, run the following commands in your shell:")
	cmd.Println(envVars)

	ripath, err := fsRepo.WriteRunInfo(ctx, envVars)
	if err != nil {
		return fmt.Errorf("writing run info to repo: %w", err)
	} else {
		cmd.Printf("A copy of these variables have been written to: %s\n", ripath)
	}
	cm.RegisterCallback(func() error {
		return os.Remove(ripath)
	})
	<-ctx.Done() // block until killed
	return nil
}

func setupLibp2p(cfg types.BacalhauConfig) (libp2pHost host.Host, peers []string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to setup libp2p node. %w", err)
		}
	}()
	privKey, err := loadLibp2pPrivKey(cfg.User.Libp2pKeyPath)
	if err != nil {
		return
	}

	libp2pHost, err = bac_libp2p.NewHost(cfg.Node.Libp2p.SwarmPort, privKey, rcmgr.DefaultResourceManager)
	if err != nil {
		return
	}

	peersAddrs, err := GetPeers(cfg)
	if err != nil {
		return
	}
	peers = make([]string, len(peersAddrs))
	for i, p := range peersAddrs {
		peers[i] = p.String()
	}
	return
}

func buildConnectCommand(ctx context.Context, nodeConfig *node.NodeConfig, ipfsConfig types.IpfsConfig) (string, error) {
	headerB := strings.Builder{}
	cmdB := strings.Builder{}
	if nodeConfig.IsRequesterNode {
		cmdB.WriteString(fmt.Sprintf("%s serve ", os.Args[0]))
		// other nodes can be just compute nodes
		// no need to spawn 1+ requester nodes
		cmdB.WriteString(fmt.Sprintf("%s=compute ",
			configflags.FlagNameForKey(types.NodeType, configflags.NodeTypeFlags...)))

		cmdB.WriteString(fmt.Sprintf("%s=%s ",
			configflags.FlagNameForKey(types.NodeNetworkType, configflags.NetworkFlags...),
			nodeConfig.NetworkConfig.Type))

		switch nodeConfig.NetworkConfig.Type {
		case models.NetworkTypeNATS:
			advertisedAddr := getPublicNATSOrchestratorURL(nodeConfig)

			headerB.WriteString("To connect a compute node to this orchestrator, run the following command in your shell:\n")
			cmdB.WriteString(fmt.Sprintf("%s=%s ",
				configflags.FlagNameForKey(types.NodeNetworkOrchestrators, configflags.NetworkFlags...),
				advertisedAddr.String(),
			))

		case models.NetworkTypeLibp2p:
			headerB.WriteString("To connect another node to this one, run the following command in your shell:\n")

			p2pAddr, err := multiaddr.NewMultiaddr("/p2p/" + nodeConfig.NetworkConfig.Libp2pHost.ID().String())
			if err != nil {
				return "", err
			}
			peerAddress := pickP2pAddress(nodeConfig.NetworkConfig.Libp2pHost.Addrs()).Encapsulate(p2pAddr).String()
			cmdB.WriteString(fmt.Sprintf("%s=%s ",
				configflags.FlagNameForKey(types.NodeLibp2pPeerConnect, configflags.Libp2pFlags...),
				peerAddress,
			))
		}

		if ipfsConfig.PrivateInternal {
			ipfsAddresses, err := nodeConfig.IPFSClient.SwarmMultiAddresses(ctx)
			if err != nil {
				return "", fmt.Errorf("error looking up IPFS addresses: %w", err)
			}

			cmdB.WriteString(fmt.Sprintf("%s ",
				configflags.FlagNameForKey(types.NodeIPFSPrivateInternal, configflags.IPFSFlags...)))

			cmdB.WriteString(fmt.Sprintf("%s=%s ",
				configflags.FlagNameForKey(types.NodeIPFSSwarmAddresses, configflags.IPFSFlags...),
				pickP2pAddress(ipfsAddresses).String(),
			))
		}
	} else {
		if nodeConfig.NetworkConfig.Type == models.NetworkTypeLibp2p {
			headerB.WriteString("Make sure there's at least one requester node in your network.")
		}
	}

	return headerB.String() + cmdB.String(), nil
}

func buildEnvVariables(
	ctx context.Context,
	cfg types.BacalhauConfig,
	nodeConfig *node.NodeConfig,
	ipfsConfig types.IpfsConfig,
) (string, error) {
	// build shell variables to connect to this node
	envVarBuilder := strings.Builder{}
	envVarBuilder.WriteString(fmt.Sprintf(
		"export %s=%s\n",
		config.KeyAsEnvVar(types.NodeClientAPIHost),
		cfg.Node.ServerAPI.Host,
	))
	envVarBuilder.WriteString(fmt.Sprintf(
		"export %s=%d\n",
		config.KeyAsEnvVar(types.NodeClientAPIPort),
		cfg.Node.ServerAPI.Port,
	))

	if nodeConfig.IsRequesterNode {
		envVarBuilder.WriteString(fmt.Sprintf(
			"export %s=%s\n",
			config.KeyAsEnvVar(types.NodeNetworkType), nodeConfig.NetworkConfig.Type,
		))

		switch nodeConfig.NetworkConfig.Type {
		case models.NetworkTypeNATS:
			envVarBuilder.WriteString(fmt.Sprintf(
				"export %s=%s\n",
				config.KeyAsEnvVar(types.NodeNetworkOrchestrators),
				getPublicNATSOrchestratorURL(nodeConfig).String(),
			))
		case models.NetworkTypeLibp2p:
			p2pAddr, err := multiaddr.NewMultiaddr("/p2p/" + nodeConfig.NetworkConfig.Libp2pHost.ID().String())
			if err != nil {
				return "", err
			}
			peerAddress := pickP2pAddress(nodeConfig.NetworkConfig.Libp2pHost.Addrs()).Encapsulate(p2pAddr).String()

			envVarBuilder.WriteString(fmt.Sprintf(
				"export %s=%s\n",
				config.KeyAsEnvVar(types.NodeLibp2pPeerConnect),
				peerAddress,
			))
		}

		if ipfsConfig.PrivateInternal {
			ipfsAddresses, err := nodeConfig.IPFSClient.SwarmMultiAddresses(ctx)
			if err != nil {
				return "", fmt.Errorf("error looking up IPFS addresses: %w", err)
			}

			envVarBuilder.WriteString(fmt.Sprintf(
				"export %s=%s\n",
				config.KeyAsEnvVar(types.NodeIPFSSwarmAddresses),
				pickP2pAddress(ipfsAddresses).String(),
			))
		}
	}

	return envVarBuilder.String(), nil
}

func getPublicNATSOrchestratorURL(nodeConfig *node.NodeConfig) *url.URL {
	orchestrator := &url.URL{
		Scheme: "nats",
		Host:   nodeConfig.NetworkConfig.AdvertisedAddress,
	}

	if nodeConfig.NetworkConfig.AdvertisedAddress == "" {
		orchestrator.Host = fmt.Sprintf("127.0.0.1:%d", nodeConfig.NetworkConfig.Port)
	}

	return orchestrator
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
func GetTLSCertificate(ctx context.Context, cfg types.BacalhauConfig, nodeConfig *node.NodeConfig) (string, string, error) {
	cert := cfg.Node.ServerAPI.TLS.ServerCertificate
	key := cfg.Node.ServerAPI.TLS.ServerKey
	if cert != "" && key != "" {
		return cert, key, nil
	}
	if cert != "" && key == "" {
		return "", "", fmt.Errorf("invalid config: TLS cert specified without corresponding private key")
	}
	if cert == "" && key != "" {
		return "", "", fmt.Errorf("invalid config: private key specified without corresponding TLS certificate")
	}
	if !nodeConfig.RequesterSelfSign {
		return "", "", nil
	}
	log.Ctx(ctx).Info().Msg("Generating self-signed certificate")
	var err error
	// If the user has not specified a private key, use their client key
	if key == "" {
		key = cfg.User.KeyPath
	}
	certFile, err := os.CreateTemp(os.TempDir(), "bacalhau_cert_*.crt")
	if err != nil {
		return "", "", errors.Wrap(err, "unable to create temporary server certificate")
	}
	defer closer.CloseWithLogOnError(certFile.Name(), certFile)

	var ips []net.IP = nil
	if ip := net.ParseIP(nodeConfig.HostAddress); ip != nil {
		ips = append(ips, ip)
	}

	if privKey, err := crypto.LoadPKCS1KeyFile(key); err != nil {
		return "", "", err
	} else if caCert, err := crypto.NewSelfSignedCertificate(privKey, false, ips); err != nil {
		return "", "", errors.Wrap(err, "failed to generate server certificate")
	} else if err = caCert.MarshalCertficate(certFile); err != nil {
		return "", "", errors.Wrap(err, "failed to write server certificate")
	}
	cert = certFile.Name()
	return cert, key, nil
}
