package ipfs

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"sync"

	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/hashicorp/go-multierror"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/phayes/freeport"
	"github.com/pkg/errors"

	"github.com/rs/zerolog/log"

	"github.com/ipfs/kubo/commands"
	"github.com/ipfs/kubo/config"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreapi"
	"github.com/ipfs/kubo/core/corehttp"
	"github.com/ipfs/kubo/core/node/libp2p"
	"github.com/ipfs/kubo/plugin/loader"
	kuboRepo "github.com/ipfs/kubo/repo"
	"github.com/ipfs/kubo/repo/fsrepo"
)

var (
	// For loading ipfs plugins once per process:
	pluginOnce sync.Once

	// Global cache of the plugin loader:
	pluginLoader *loader.PluginLoader
)

const (
	// The default size of a node's repo keypair.
	defaultKeypairSize = 2048
)

// Node is a wrapper around an in-process IPFS node that can be used to
// interact with the IPFS network without requiring an `ipfs` binary.
type Node struct {
	api      icore.CoreAPI
	ipfsNode *core.IpfsNode

	// Mode is the mode the ipfs node was created in.
	Mode NodeMode

	// RepoPath is the path to the ipfs node's data repository.
	RepoPath string

	// APIPort is the port that the node's ipfs API is listening on.
	APIPort int

	// SwarmPort is the port that the node's ipfs swarm is listening on.
	SwarmPort int
}

// NodeMode configures how the node treats the public IPFS network.
type NodeMode int

const (
	// ModeDefault is the default node mode, which uses an IPFS repo backed
	// by the `flatfs` datastore, and connects to the public IPFS network.
	ModeDefault NodeMode = iota

	// ModeLocal is a node mode that uses an IPFS repo backed by the `flatfs`
	// datastore and ignores the public IPFS network completely, for setting
	// up test environments without polluting the public IPFS nodes.
	ModeLocal
)

// Config contains configuration for the IPFS node.
type Config struct {
	// PeerAddrs is a list of additional IPFS node multiaddrs to use as
	// peers. By default, the IPFS node will connect to whatever nodes are
	// specified by its mode.
	PeerAddrs []string

	// Mode configures the node's default settings.
	Mode NodeMode

	// KeypairSize is the number of bits to use for the node's repo keypair. If
	// nil, then a default value of 2048 is used.
	KeypairSize int
}

func (cfg *Config) getKeypairSize() int {
	if cfg.KeypairSize == 0 {
		return defaultKeypairSize
	}

	return cfg.KeypairSize
}

func (cfg *Config) getMode() NodeMode {
	return cfg.Mode
}

func (cfg *Config) getPeerAddrs() []string {
	return cfg.PeerAddrs
}

// NewNode creates a new IPFS node in default mode, which creates an IPFS
// repo in a temporary directory, uses the public libp2p nodes as peers and
// generates a repo keypair with 2048 bits.
func NewNode(ctx context.Context, cm *system.CleanupManager, peerAddrs []string) (*Node, error) {
	return newNode(ctx, cm, peerAddrs, ModeDefault)
}

// NewLocalNode creates a new local IPFS node in local mode, which can be used
// to create test environments without polluting the public IPFS nodes.
func NewLocalNode(ctx context.Context, cm *system.CleanupManager, peerAddrs []string) (*Node, error) {
	return newNode(ctx, cm, peerAddrs, ModeLocal)
}

func newNode(ctx context.Context, cm *system.CleanupManager, peerAddrs []string, mode NodeMode) (*Node, error) {
	// filter out any empty peer addresses
	filteredPeerAddrs := make([]string, 0, len(peerAddrs))
	for _, addr := range peerAddrs {
		if addr != "" {
			filteredPeerAddrs = append(filteredPeerAddrs, addr)
		}
	}
	return tryCreateNode(ctx, cm, Config{
		Mode:      mode,
		PeerAddrs: filteredPeerAddrs,
	})
}

func tryCreateNode(ctx context.Context, cm *system.CleanupManager, cfg Config) (*Node, error) {
	// Starting up an IPFS node can have issues as there's a race between finding a free port and getting the listener
	// running on that port (e.g. find the port, write the config file, save the file, start up IPFS, then start the listener)
	attempts := 3
	var err error
	for i := 0; i < attempts; i++ {
		var ipfsNode *Node
		ipfsNode, err = newNodeWithConfig(ctx, cm, cfg)
		if err != nil {
			if errors.Is(err, addressInUseError) {
				log.Ctx(ctx).Debug().Err(err).Msg("Failed to start up node as port was already in use")
				continue
			}
			return nil, err
		}

		return ipfsNode, nil
	}
	return nil, err
}

// newNodeWithConfig creates a new IPFS node with the given configuration.
// NOTE: use NewNode() or NewLocalNode() unless you know what you're doing.
func newNodeWithConfig(ctx context.Context, cm *system.CleanupManager, cfg Config) (*Node, error) {
	var err error
	pluginOnce.Do(func() {
		err = loadPlugins(cm)
	})
	if err != nil {
		return nil, err
	}

	api, ipfsNode, repoPath, err := createNode(ctx, cm, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create ipfs node: %w", err)
	}
	defer func() {
		if err != nil {
			_ = ipfsNode.Close()
		}
	}()

	if err = connectToPeers(ctx, api, ipfsNode, cfg.getPeerAddrs()); err != nil {
		log.Ctx(ctx).Error().Msgf("ipfs node failed to connect to peers: %s", err)
	}

	if err = serveAPI(cm, ipfsNode, repoPath); err != nil {
		return nil, fmt.Errorf("failed to serve API: %w", err)
	}

	// Fetch useful info from the newly initialized node:
	nodeCfg, err := ipfsNode.Repo.Config()
	if err != nil {
		return nil, fmt.Errorf("failed to get repo config: %w", err)
	}

	var apiPort int
	if len(nodeCfg.Addresses.API) > 0 {
		apiPort, err = getTCPPort(nodeCfg.Addresses.API[0])
		if err != nil {
			return nil, fmt.Errorf("failed to parse api port: %w", err)
		}
	}

	var swarmPort int
	if len(nodeCfg.Addresses.Swarm) > 0 {
		swarmPort, err = getTCPPort(nodeCfg.Addresses.Swarm[0])
		if err != nil {
			return nil, fmt.Errorf("failed to parse swarm port: %w", err)
		}
	}

	n := Node{
		api:       api,
		ipfsNode:  ipfsNode,
		Mode:      cfg.getMode(),
		RepoPath:  repoPath,
		APIPort:   apiPort,
		SwarmPort: swarmPort,
	}

	cm.RegisterCallbackWithContext(n.Close)

	// Log details so that user can connect to the new node:
	log.Ctx(ctx).Trace().Msgf("IPFS node created with ID: %s", ipfsNode.Identity)
	n.LogDetails()

	return &n, nil
}

// ID returns the node's ipfs ID.
func (n *Node) ID() string {
	return n.ipfsNode.Identity.String()
}

// APIAddresses returns the node's api addresses.
func (n *Node) APIAddresses() ([]string, error) {
	cfg, err := n.ipfsNode.Repo.Config()
	if err != nil {
		return nil, fmt.Errorf("failed to get repo config: %w", err)
	}

	var res []string
	for _, addr := range cfg.Addresses.API {
		res = append(res, fmt.Sprintf("%s/p2p/%s", addr, n.ID()))
	}

	return res, nil
}

// SwarmAddresses returns the node's swarm addresses.
func (n *Node) SwarmAddresses() ([]string, error) {
	cfg, err := n.ipfsNode.Repo.Config()
	if err != nil {
		return nil, fmt.Errorf("failed to get repo config: %w", err)
	}

	var res []string
	for _, addr := range cfg.Addresses.Swarm {
		res = append(res, fmt.Sprintf("%s/p2p/%s", addr, n.ID()))
	}

	return res, nil
}

// LogDetails logs connection details for the node's swarm and API servers.
func (n *Node) LogDetails() {
	apiAddrs, err := n.APIAddresses()
	if err != nil {
		log.Debug().Msgf("error fetching api addresses: %s", err)
		return
	}

	var swarmAddrs []string
	swarmAddrs, err = n.SwarmAddresses()
	if err != nil {
		log.Debug().Msgf("error fetching swarm addresses: %s", err)
	}

	id := n.ID()
	for _, apiAddr := range apiAddrs {
		log.Trace().Msgf("IPFS node %s listening for API on: %s", id, apiAddr)
	}
	for _, swarmAddr := range swarmAddrs {
		log.Trace().Msgf("IPFS node %s listening for swarm on: %s", id, swarmAddr)
	}
}

// Client returns an API client for interacting with the node.
func (n *Node) Client() Client {
	return NewClient(n.api)
}

func (n *Node) Close(ctx context.Context) error {
	log.Ctx(ctx).Debug().Msgf("Closing IPFS node %s", n.ID())
	var errs *multierror.Error
	if n.ipfsNode != nil {
		errs = multierror.Append(errs, n.ipfsNode.Close())

		// We need to make sure we close the repo before we delete the disk contents as this will cause IPFS to print out messages about how
		// 'flatfs could not store final value of disk usage to file', which is both annoying and can cause test flakes
		// as the message can be written just after the test has finished but before the repo has been told by node
		// that it's supposed to shut down.
		if n.ipfsNode.Repo != nil {
			if err := n.ipfsNode.Repo.Close(); err != nil { //nolint:govet
				errs = multierror.Append(errs, fmt.Errorf("failed to close repo: %w", err))
			}
		}
	}

	if n.RepoPath != "" {
		if err := os.RemoveAll(n.RepoPath); err != nil { //nolint:govet
			errs = multierror.Append(errs, fmt.Errorf("failed to clean up repo directory: %w", err))
		}
	}
	return errs.ErrorOrNil()
}

// createNode spawns a new IPFS node using a temporary repo path.
func createNode(ctx context.Context, _ *system.CleanupManager, cfg Config) (icore.CoreAPI, *core.IpfsNode, string, error) {
	repoPath, err := os.MkdirTemp("", "ipfs-tmp")
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to create repo dir: %w", err)
	}

	var repo kuboRepo.Repo
	if err = createRepo(repoPath, cfg); err != nil {
		return nil, nil, "", fmt.Errorf("failed to create repo: %w", err)
	}

	repo, err = fsrepo.Open(repoPath)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to open temp repo: %w", err)
	}

	nodeOptions := &core.BuildCfg{
		Repo:    repo,
		Online:  true,
		Routing: libp2p.DHTClientOption,
	}

	node, err := core.NewNode(ctx, nodeOptions)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to create node: %w", err)
	}

	api, err := coreapi.NewCoreAPI(node)
	return api, node, repoPath, err
}

// serveAPI starts a new API server for the node on the given address.
func serveAPI(cm *system.CleanupManager, node *core.IpfsNode, repoPath string) error {
	cfg, err := node.Repo.Config()
	if err != nil {
		return fmt.Errorf("failed to get repo config: %w", err)
	}

	var listeners []manet.Listener
	for _, addr := range cfg.Addresses.API {
		maddr, err := ma.NewMultiaddr(addr)
		if err != nil {
			return fmt.Errorf("failed to parse multiaddr: %w", err)
		}

		listener, err := manet.Listen(maddr)
		if err != nil {
			return fmt.Errorf("failed to listen on api multiaddr: %w", err)
		}

		cm.RegisterCallback(func() error {
			if err := listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
				return errors.Wrap(err, "error shutting down IPFS listener")
			}
			return nil
		})

		listeners = append(listeners, listener)
	}

	// We need to construct a commands.Context in order to use the node APIs:
	cmdContext := commands.Context{
		ReqLog:     &commands.ReqLog{},
		Plugins:    pluginLoader,
		ConfigRoot: repoPath,
		ConstructNode: func() (n *core.IpfsNode, err error) {
			return node, nil
		},
	}

	// Options determine which functionality the API should include:
	var opts = []corehttp.ServeOption{
		corehttp.VersionOption(),
		corehttp.GatewayOption(false),
		corehttp.WebUIOption,
		corehttp.CommandsOption(cmdContext),
	}

	for _, listener := range listeners {
		// NOTE: this is not critical, but we should log for debugging
		go func(listener manet.Listener) {
			if err := corehttp.Serve(node, manet.NetListener(listener), opts...); err != nil {
				log.Debug().Msgf("node '%s' failed to serve ipfs api: %s", node.Identity, err)
			}
		}(listener)
	}

	return nil
}

// connectToPeers connects the node to a list of IPFS bootstrap peers.
// event though we have Peering enabled, some test scenarios relies on the node being eagerly connected to the peers
func connectToPeers(ctx context.Context, api icore.CoreAPI, node *core.IpfsNode, peerAddrs []string) error {
	log.Ctx(ctx).Debug().Msgf("IPFS node %s has current peers: %v", node.Identity, node.Peerstore.Peers())
	log.Ctx(ctx).Debug().Msgf("IPFS node %s is connecting to new peers: %v", node.Identity, peerAddrs)

	// Parse the bootstrap node multiaddrs and fetch their IPFS peer info:
	peerInfos, err := ParsePeersString(peerAddrs)
	if err != nil {
		return err
	}

	// Bootstrap the node's list of peers:
	var anyErr error
	var wg sync.WaitGroup
	wg.Add(len(peerInfos))
	for _, peerInfo := range peerInfos {
		go func(peerInfo peer.AddrInfo) {
			defer wg.Done()
			if err := api.Swarm().Connect(ctx, peerInfo); err != nil {
				anyErr = err
				log.Ctx(ctx).Debug().Msgf(
					"failed to connect to ipfs peer %s, skipping: %s",
					peerInfo.ID, err)
			}
		}(peerInfo)
	}

	wg.Wait()
	return anyErr
}

// createRepo creates an IPFS repository in a given directory.
func createRepo(path string, nodeConfig Config) error {
	cfg, err := config.Init(io.Discard, nodeConfig.getKeypairSize())
	if err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	profile := "flatfs"
	if nodeConfig.getMode() == ModeLocal {
		profile = "test"
	}

	transformer, ok := config.Profiles[profile]
	if !ok {
		return fmt.Errorf("invalid configuration profile: %s", profile)
	}
	if err := transformer.Transform(cfg); err != nil { //nolint: govet
		return err
	}

	var apiPort int
	apiPort, err = freeport.GetFreePort()
	if err != nil {
		return fmt.Errorf("could not create port for api: %w", err)
	}

	// If we're in local mode, then we need to manually change the config to
	// serve an IPFS swarm client on some local port:
	if nodeConfig.getMode() == ModeLocal {
		var gatewayPort int
		gatewayPort, err = freeport.GetFreePort()
		if err != nil {
			return fmt.Errorf("could not create port for gateway: %w", err)
		}

		var swarmPort int
		swarmPort, err = freeport.GetFreePort()
		if err != nil {
			return fmt.Errorf("could not create port for swarm: %w", err)
		}

		cfg.AutoNAT.ServiceMode = config.AutoNATServiceDisabled
		cfg.Swarm.EnableHolePunching = config.False
		cfg.Swarm.DisableNatPortMap = true
		cfg.Swarm.RelayClient.Enabled = config.False
		cfg.Swarm.RelayService.Enabled = config.False
		cfg.Swarm.Transports.Network.Relay = config.False
		cfg.Discovery.MDNS.Enabled = false
		cfg.Addresses.Gateway = []string{
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", gatewayPort),
		}
		cfg.Addresses.API = []string{
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", apiPort),
		}
		cfg.Addresses.Swarm = []string{
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", swarmPort),
		}
	} else {
		cfg.Addresses.API = []string{
			fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", apiPort),
		}
	}

	// establish peering with the passed nodes. This is different than bootstrapping or manually connecting to peers,
	//and kubo will create sticky connections with these nodes and reconnect if the connection is lost
	// https://github.com/ipfs/kubo/blob/master/docs/config.md#peering
	swarmPeers, err := ParsePeersString(nodeConfig.getPeerAddrs())
	if err != nil {
		return fmt.Errorf("failed to parse peer addresses: %w", err)
	}
	cfg.Peering = config.Peering{
		Peers: swarmPeers,
	}

	err = fsrepo.Init(path, cfg)
	if err != nil {
		return fmt.Errorf("failed to init ipfs repo: %w", err)
	}

	return nil
}

// loadPlugins initializes and injects the standard set of ipfs plugins.
func loadPlugins(cm *system.CleanupManager) error {
	plugins, err := loader.NewPluginLoader("")
	if err != nil {
		return fmt.Errorf("error loading plugins: %s", err)
	}

	if err := plugins.Initialize(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	if err := plugins.Inject(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	// Set the global cache so we can use it in the ipfs daemon:
	pluginLoader = plugins
	cm.RegisterCallback(plugins.Close)
	return nil
}

// getTCPPort returns the tcp port in a multiaddress.
func getTCPPort(addr string) (int, error) {
	maddr, err := ma.NewMultiaddr(addr)
	if err != nil {
		return 0, err
	}

	p, err := maddr.ValueForProtocol(ma.P_TCP)
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(p)
}
