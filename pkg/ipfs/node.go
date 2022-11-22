package ipfs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"sync"

	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/hashicorp/go-multierror"
	icore "github.com/ipfs/interface-go-ipfs-core"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/phayes/freeport"

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
	"github.com/libp2p/go-libp2p/core/peer"
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
	api    icore.CoreAPI
	node   *core.IpfsNode
	cancel context.CancelFunc

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
	KeypairSize *int
}

func (cfg *Config) getKeypairSize() int {
	if cfg.KeypairSize == nil {
		return defaultKeypairSize
	}

	return *cfg.KeypairSize
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
	// filter out any empty peer addresses
	filteredPeerAddrs := make([]string, 0, len(peerAddrs))
	for _, addr := range peerAddrs {
		if addr != "" {
			filteredPeerAddrs = append(filteredPeerAddrs, addr)
		}
	}
	return tryCreateNode(ctx, cm, Config{
		Mode:      ModeDefault,
		PeerAddrs: filteredPeerAddrs,
	})
}

// NewLocalNode creates a new local IPFS node in local mode, which can be used
// to create test environments without polluting the public IPFS nodes.
func NewLocalNode(ctx context.Context, cm *system.CleanupManager, peerAddrs []string) (*Node, error) {
	return tryCreateNode(ctx, cm, Config{
		Mode:      ModeLocal,
		PeerAddrs: peerAddrs,
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
		err = loadPlugins()
	})
	if err != nil {
		return nil, err
	}

	// go-ipfs uses contexts for lifecycle management:
	ctx, cancel := context.WithCancel(ctx)
	cm.RegisterCallback(func() error {
		cancel()
		return nil
	})

	api, node, repoPath, err := createNode(ctx, cm, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create ipfs node: %w", err)
	}

	if err = connectToPeers(ctx, api, node, cfg.getPeerAddrs()); err != nil {
		log.Error().Msgf("ipfs node failed to connect to peers: %s", err)
	}

	if err = serveAPI(cm, node, repoPath); err != nil {
		return nil, fmt.Errorf("failed to serve API: %w", err)
	}

	// Fetch useful info from the newly initialized node:
	nodeCfg, err := node.Repo.Config()
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
		api:    api,
		node:   node,
		cancel: cancel,

		Mode:      cfg.getMode(),
		RepoPath:  repoPath,
		APIPort:   apiPort,
		SwarmPort: swarmPort,
	}

	// Log details so that user can connect to the new node:
	log.Trace().Msgf("IPFS node created with ID: %s", node.Identity)
	n.LogDetails()

	return &n, nil
}

// ID returns the node's ipfs ID.
func (n *Node) ID() string {
	return n.node.Identity.String()
}

// APIAddresses returns the node's api addresses.
func (n *Node) APIAddresses() ([]string, error) {
	cfg, err := n.node.Repo.Config()
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
	cfg, err := n.node.Repo.Config()
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
func (n *Node) Client() (*Client, error) {
	addrs, err := n.APIAddresses()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch api addresses: %w", err)
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("error creating client: node has no available api addresses")
	}

	return NewClient(addrs[0])
}

// createNode spawns a new IPFS node using a temporary repo path.
func createNode(ctx context.Context, cm *system.CleanupManager, cfg Config) (icore.CoreAPI, *core.IpfsNode, string, error) {
	repoPath, err := os.MkdirTemp("", "ipfs-tmp")
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to create repo dir: %w", err)
	}

	var repo kuboRepo.Repo
	cm.RegisterCallback(func() error {
		var errs error
		// We need to make sure we close the repo before we delete the disk contents as this will cause IPFS to print out messages about how
		// 'flatfs could not store final value of disk usage to file', which is both annoying and can cause test flakes
		// as the message can be written just after the test has finished but before the repo has been told by node
		// that it's supposed to shut down.
		if repo != nil {
			if err := repo.Close(); err != nil { //nolint:govet
				errs = multierror.Append(errs, fmt.Errorf("failed to close repo: %w", err))
			}
		}
		if err := os.RemoveAll(repoPath); err != nil { //nolint:govet
			errs = multierror.Append(errs, fmt.Errorf("failed to clean up repo directory: %w", err))
		}
		return errs
	})

	if err = createRepo(repoPath, cfg.getMode(), cfg.getKeypairSize()); err != nil {
		return nil, nil, "", fmt.Errorf("failed to create repo: %w", err)
	}

	repo, err = fsrepo.Open(repoPath)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to open temp repo: %w", err)
	}

	nodeOptions := &core.BuildCfg{
		Repo:    repo,
		Online:  true,
		Routing: libp2p.DHTOption,
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
		go func(listener manet.Listener) {
			cm.RegisterCallback(func() error {
				if err := listener.Close(); err != nil {
					if !errors.Is(err, net.ErrClosed) {
						return fmt.Errorf("problem when shutting down IPFS listener: %w", err)
					}

					// I'm fairly sure this error occurs because the listener is getting closed twice
					// once in this callback and again when `corehttp.Serve` returns (it has a defer statement).
					// `corehttp.Serve` looks like it'll return when the context passed into the node on creation gets
					// closed.
					log.Debug().Err(err).Msg("Error occurred when trying to shut down listener")
				}
				return nil
			})

			// NOTE: this is not critical, but we should log for debugging
			if err := corehttp.Serve(node, manet.NetListener(listener), opts...); err != nil {
				log.Debug().Msgf("node '%s' failed to serve ipfs api: %s", node.Identity, err)
			}
		}(listener)
	}

	return nil
}

// connectToPeers connects the node to a list of IPFS bootstrap peers.
func connectToPeers(ctx context.Context, api icore.CoreAPI, node *core.IpfsNode, peerAddrs []string) error {
	log.Debug().Msgf("IPFS node %s has current peers: %v", node.Identity, node.Peerstore.Peers())
	log.Debug().Msgf("IPFS node %s is connecting to new peers: %v", node.Identity, peerAddrs)

	// Parse the bootstrap node multiaddrs and fetch their IPFS peer info:
	peerInfos := make(map[peer.ID]*peer.AddrInfo)
	for _, addrStr := range peerAddrs {
		addr, err := ma.NewMultiaddr(addrStr)
		if err != nil {
			return err
		}

		pii, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			return err
		}

		peerInfos[pii.ID] = pii
	}

	// Bootstrap the node's list of peers:
	var anyErr error
	var wg sync.WaitGroup
	wg.Add(len(peerInfos))
	for _, peerInfo := range peerInfos {
		go func(peerInfo *peer.AddrInfo) {
			defer wg.Done()
			if err := api.Swarm().Connect(ctx, *peerInfo); err != nil {
				anyErr = err
				log.Debug().Msgf(
					"failed to connect to ipfs peer %s, skipping: %s",
					peerInfo.ID, err)
			}
		}(peerInfo)
	}

	wg.Wait()
	return anyErr
}

// createRepo creates an IPFS repository in a given directory.
func createRepo(path string, mode NodeMode, keypairSize int) error {
	cfg, err := config.Init(io.Discard, keypairSize)
	if err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	profile := "flatfs"
	if mode == ModeLocal {
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
	if mode == ModeLocal {
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
		cfg.Addresses.Announce = []string{
			fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", swarmPort),
		}
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

	err = fsrepo.Init(path, cfg)
	if err != nil {
		return fmt.Errorf("failed to init ipfs repo: %w", err)
	}

	return nil
}

// loadPlugins initializes and injects the standard set of ipfs plugins.
func loadPlugins() error {
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
