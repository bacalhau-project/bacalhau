package serve

import (
	"fmt"
	"strings"

	"github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/configflags"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/nodefx"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
	"github.com/bacalhau-project/bacalhau/pkg/system"
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
		panic("TODO")
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
	// define the serve command
	serveCmd := &cobra.Command{
		Use:     "serve",
		Short:   "Start the bacalhau compute node",
		Long:    serveLong,
		Example: serveExample,
	}

	// define flags supported on the serve command
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

	// register flags on the command
	if err := configflags.RegisterFlags(serveCmd, serveFlags); err != nil {
		// a failure here indicates a developer error in defining flags.
		util.Fatal(serveCmd, err, 1)
	}

	// bind the server flags to the configuration s.t. we use values in order of precedence based on:
	// 1. CLI flag.
	// 2. Environment Variable.
	// 3. Config File.
	// 4. Defaults.
	serveCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		return configflags.BindFlagsWithViper(cmd, cfg.User(), serveFlags)
	}

	// define the run method which accepts the config we created in the above steps.
	serveCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return serve(cmd, cfg)
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

	defaultConfig, err := cfg.Current()
	if err != nil {
		panic(err)
	}
	_ = defaultConfig
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
