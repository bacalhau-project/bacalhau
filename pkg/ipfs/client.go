package ipfs

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/filecoin-project/bacalhau/pkg/system"
	files "github.com/ipfs/go-ipfs-files"
	icore "github.com/ipfs/interface-go-ipfs-core"
	icorepath "github.com/ipfs/interface-go-ipfs-core/path"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/phayes/freeport"

	"github.com/rs/zerolog/log"

	"github.com/ipfs/go-ipfs/config"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/core/node/libp2p"
	"github.com/ipfs/go-ipfs/plugin/loader"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/libp2p/go-libp2p-core/peer"
)

var (
	// For loading ipfs plugins once per process:
	pluginOnce sync.Once

	// The default list of nodes to use as peers:
	defaultBootstrapNodes = []string{
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
	}
)

const (
	// The default size of a node's repo keypair.
	defaultKeypairSize = 2048
)

// Client is a wrapper around an in-process IPFS node that can be used to
// interact with the IPFS network without requiring an `ipfs` binary.
type Client struct {
	api    icore.CoreAPI
	node   *core.IpfsNode
	cancel context.CancelFunc
	Mode   ClientMode
}

// ClientMode configures how the client treats the public IPFS network.
type ClientMode int

const (
	// ModeDefault is the default client mode, which uses an IPFS repo backed
	// by the `flatfs` datastore, and connects to the public IPFS network.
	ModeDefault ClientMode = iota

	// ModeLocal is a client mode that uses an IPFS repo backed by the `flatfs`
	// datastore and ignores the public IPFS network completely, for setting
	// up test environments without polluting the public IPFS nodes.
	ModeLocal
)

// Config contains configuration for the IPFS node.
type Config struct {
	// RepoPath is the path to the node's IPFS repository. If nil, then a
	// random temporary directory is initialized as the node's repository.
	RepoPath *string

	// BootstrapNodes is a list of additional IPFS node multiaddrs to use as
	// peers. By default, the IPFS node will connect to whatever nodes are
	// listed in its profile, or the public IPFS nodes if no profile is
	// specified.
	BootstrapNodes []string

	// Mode configures how the client treats the public IPFS network. If nil,
	// then ModeDefault is used, which connects to the public IPFS network.
	Mode ClientMode

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

func (cfg *Config) getRepoPath() (string, error) {
	if cfg.RepoPath == nil {
		path, err := os.MkdirTemp("", "ipfs-tmp")
		if err != nil {
			return "", fmt.Errorf("failed to create temp dir: %w", err)
		}

		return path, nil
	}

	return *cfg.RepoPath, nil
}

func (cfg *Config) getMode() ClientMode {
	return cfg.Mode
}

func (cfg *Config) getBootstrapNodes() []string {
	if cfg.BootstrapNodes == nil {
		return defaultBootstrapNodes
	}

	return cfg.BootstrapNodes
}

// NewClient creates a new IPFS client in default mode, which creates an IPFS
// repo in a temporary directory, uses the public libp2p nodes as peers and
// generates a repo keypair with 2048 bits.
func NewClient(cm *system.CleanupManager, peers []string) (*Client, error) {
	return NewClientWithConfig(cm, Config{
		BootstrapNodes: peers,
	})
}

// NewLocalClient creates a new local IPFS client in local mode, which can
// be used to create test environments without polluting the public IPFS nodes.
func NewLocalClient(cm *system.CleanupManager, peers []string) (*Client, error) {
	log.Debug().Msgf("%v", peers)
	return NewClientWithConfig(cm, Config{
		Mode:           ModeLocal,
		BootstrapNodes: peers,
	})
}

// NewClientWithConfig creates a new IPFS client with the given configuration.
// NOTE: use NewClient() or NewLocalClient() unless you know what you're doing.
func NewClientWithConfig(cm *system.CleanupManager, cfg Config) (*Client, error) {
	var err error
	pluginOnce.Do(func() {
		err = loadPlugins()
	})
	if err != nil {
		return nil, err
	}

	// go-ipfs uses contexts for lifecycle management:
	ctx, cancel := context.WithCancel(context.Background())
	cm.RegisterCallback(func() error {
		cancel()
		return nil
	})

	api, node, err := createNode(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create ipfs node: %w", err)
	}
	log.Debug().Msgf("IPFS node created with ID: %s", node.Identity)

	if err := connectToPeers(ctx, api, node, cfg.getBootstrapNodes()); err != nil {
		log.Error().Msgf("ipfs client failed to connect to peers: %s", err)
	}

	return &Client{
		api:    api,
		node:   node,
		cancel: cancel,
	}, nil
}

// ID returns the client's ipfs node ID.
func (cl *Client) ID() string {
	return cl.node.Identity.String()
}

// P2pAddrs returns the client's libp2p addresses.
func (cl *Client) P2pAddrs() ([]string, error) {
	cfg, err := cl.node.Repo.Config()
	if err != nil {
		return nil, fmt.Errorf("failed to get repo config: %w", err)
	}

	var res []string
	for _, addr := range cfg.Addresses.Swarm {
		res = append(res, fmt.Sprintf("%s/p2p/%s", addr, cl.ID()))
	}

	return res, nil
}

// Get fetches a file or directory from the IPFS network.
func (cl *Client) Get(ctx context.Context, cid, outputPath string) error {
	node, err := cl.api.Unixfs().Get(ctx, icorepath.New(cid))
	if err != nil {
		return fmt.Errorf("failed to get file '%s': %w", cid, err)
	}

	return files.WriteTo(node, outputPath)
}

// Put uploads a file or directory to the IPFS network.
func (cl *Client) Put(ctx context.Context, inputPath string) (string, error) {
	st, err := os.Stat(inputPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat file '%s': %w", inputPath, err)
	}

	node, err := files.NewSerialFile(inputPath, false, st)
	if err != nil {
		return "", fmt.Errorf("failed to create ipfs node: %w", err)
	}

	cid, err := cl.api.Unixfs().Add(ctx, node)
	if err != nil {
		return "", fmt.Errorf("failed to add file '%s': %w", inputPath, err)
	}

	return cid.String(), nil
}

// connectToPeers connects the node to a list of IPFS bootstrap peers.
func connectToPeers(ctx context.Context, api icore.CoreAPI, node *core.IpfsNode, bootstrapNodes []string) error {
	log.Debug().Msgf("IPFS node %s has current peers: %v", node.Identity, node.Peerstore.Peers())
	log.Debug().Msgf("IPFS node %s is connecting to new peers: %v", node.Identity, bootstrapNodes)

	// Parse the bootstrap node multiaddrs and fetch their IPFS peer info:
	peerInfos := make(map[peer.ID]*peer.AddrInfo)
	for _, addrStr := range bootstrapNodes {
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
	var wg sync.WaitGroup
	wg.Add(len(peerInfos))
	for _, peerInfo := range peerInfos {
		go func(peerInfo *peer.AddrInfo) {
			defer wg.Done()
			if err := api.Swarm().Connect(ctx, *peerInfo); err != nil {
				log.Debug().Msgf(
					"failed to connect to ipfs peer %s, skipping: %s",
					peerInfo.ID, err)
			}
		}(peerInfo)
	}

	wg.Wait()
	return nil
}

// createNode spawns a new IPFS node using a temporary repo path.
func createNode(ctx context.Context, cfg Config) (icore.CoreAPI, *core.IpfsNode, error) {
	repoPath, err := cfg.getRepoPath()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create repo dir: %w", err)
	}

	if err = createRepo(repoPath, cfg.getMode(), cfg.getKeypairSize()); err != nil {
		return nil, nil, fmt.Errorf("failed to create repo: %w", err)
	}

	repo, err := fsrepo.Open(repoPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open temp repo: %w", err)
	}

	nodeOptions := &core.BuildCfg{
		Repo:    repo,
		Online:  true,
		Routing: libp2p.DHTOption,
	}

	node, err := core.NewNode(ctx, nodeOptions)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create node: %w", err)
	}

	api, err := coreapi.NewCoreAPI(node)
	return api, node, err
}

// createRepo creates an IPFS repository in a given directory.
func createRepo(path string, mode ClientMode, keypairSize int) error {
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
	if err := transformer.Transform(cfg); err != nil { // nolint: govet
		return err
	}

	// If we're in local mode, then we need to manually change the config to
	// serve an IPFS swarm client on some local port:
	if mode == ModeLocal {
		var gatewayPort int
		gatewayPort, err = freeport.GetFreePort()
		if err != nil {
			return fmt.Errorf("could not create port for gateway: %w", err)
		}

		var apiPort int
		apiPort, err = freeport.GetFreePort()
		if err != nil {
			return fmt.Errorf("could not create port for api: %w", err)
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
			fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", gatewayPort),
		}
		cfg.Addresses.API = []string{
			fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", apiPort),
		}
		cfg.Addresses.Swarm = []string{
			fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", swarmPort),
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

	return nil
}
