package serve

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/configflags"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/lib/crypto"
	bac_libp2p "github.com/bacalhau-project/bacalhau/pkg/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/libp2p/rcmgr"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/nodefx"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
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
		// bootstrap peer list from their config file.
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

func NewCmd(cfg *config.Config) *cobra.Command {
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

	// cfg := config.New(config.ForEnvironment())

	serveCmd := &cobra.Command{
		Use:     "serve",
		Short:   "Start the bacalhau compute node",
		Long:    serveLong,
		Example: serveExample,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// bind flags for this command to the viper instance used to configure bacalhau from command flags.
			return configflags.BindFlagsWithViper(cmd, cfg.Viper(), serveFlags)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return serve(cmd, cfg)
		},
	}

	if err := configflags.RegisterFlags(serveCmd, serveFlags); err != nil {
		util.Fatal(serveCmd, err, 1)
	}
	return serveCmd
}

//nolint:funlen,gocyclo
func serve(cmd *cobra.Command, cfg *config.Config) error {
	ctx := cmd.Context()
	cm := util.GetCleanupManager(ctx)

	repoDir, err := cmd.Root().PersistentFlags().GetString("repo")
	if err != nil {
		return err
	}
	fsRepo, err := setup.SetupBacalhauRepo(repoDir, cfg)
	if err != nil {
		return err
	}

	// configure node type
	isRequesterNode, isComputeNode, err := getNodeType(cfg)
	if err != nil {
		return err
	}

	// Establishing IPFS connection
	ipfsConfig, err := getIPFSConfig(cfg)
	if err != nil {
		return err
	}

	ipfsClient, err := SetupIPFSClient(ctx, cm, ipfsConfig)
	if err != nil {
		return err
	}

	// Create node
	standardNode, shutdown, err := nodefx.New(ctx,
		nodefx.Repo(fsRepo),
		nodefx.IPFSClient(ipfsClient),
		nodefx.Config(cfg),
		nodefx.ComputeNode(isComputeNode),
		nodefx.RequesterNode(isRequesterNode),
	)
	if err != nil {
		return fmt.Errorf("error creating node: %w", err)
	}
	defer func() {
		if err := shutdown(); err != nil {
			log.Err(err).Msg("shutdown unsuccessful")
		}
	}()

	_ = standardNode
	<-ctx.Done() // block until killed
	return nil
}

func setupLibp2p() (libp2pHost host.Host, peers []string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to setup libp2p node. %w", err)
		}
	}()
	libp2pCfg, err := config.GetLibp2pConfig()
	if err != nil {
		return
	}

	privKey, err := config.GetLibp2pPrivKey()
	if err != nil {
		return
	}

	libp2pHost, err = bac_libp2p.NewHost(libp2pCfg.SwarmPort, privKey, rcmgr.DefaultResourceManager)
	if err != nil {
		return
	}

	peersAddrs, err := GetPeers(libp2pCfg.PeerConnect)
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

func buildEnvVariables(ctx context.Context, nodeConfig *node.NodeConfig, ipfsConfig types.IpfsConfig) (string, error) {
	// build shell variables to connect to this node
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
func GetTLSCertificate(ctx context.Context, nodeConfig *node.NodeConfig) (string, string, error) {
	cert, key := config.GetRequesterCertificateSettings()
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
		key, err = config.Get[string](types.UserKeyPath)
		if err != nil {
			return "", "", err
		}
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
