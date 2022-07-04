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
}

// Config contains configuration for the IPFS node.
type Config struct {
	// RepoPath is the path to the node's IPFS repository. If nil, then a
	// random temporary directory is used as the node's repository.
	RepoPath *string

	// BootstrapNodes is a list of IPFS node multiaddrs to use as peers. If
	// nil, then the public bootstrap.libp2p.io nodes are used. Note that the
	// list of bootstrap nodes may be non-nil and empty to signal that the
	// node connect to no peers.
	BootstrapNodes []string

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

func (cfg *Config) getBootstrapNodes() []string {
	if cfg.BootstrapNodes == nil {
		return defaultBootstrapNodes
	}

	return cfg.BootstrapNodes
}

// NewDefaultClient creates a new IPFS client with the default configuration,
// which creates an IPFS repo in a temporary directory, uses the public libp2p
// nodes as peers and generates a repo keypair with 2048 bits.
func NewDefaultClient(cm *system.CleanupManager) (*Client, error) {
	return NewClient(cm, Config{})
}

// NewClient creates a new IPFS client with the given configuration.
func NewClient(cm *system.CleanupManager, cfg Config) (*Client, error) {
	var err error
	pluginOnce.Do(func() {
		err = loadPlugins()
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	cm.RegisterCallback(func() error {
		cancel()
		return nil
	})

	api, node, err := createNode(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create ipfs node: %w", err)
	}

	// Connecting to bootstrap peers can be done asynchronously:
	go func() {
		if err := connectToPeers(ctx, api, cfg.getBootstrapNodes()); err != nil {
			log.Error().Msgf("ipfs client failed to connect to peers: %s", err)
		}
	}()

	return &Client{
		api:    api,
		node:   node,
		cancel: cancel,
	}, nil
}

// Multiaddr returns the client's ipfs node multiaddress.
func (cl *Client) Multiaddr() string {
	return fmt.Sprintf("/ip4/127.0.0.1/p2p/%s", cl.node.Identity)
}

// Get fetches a file from the IPFS network.
func (cl *Client) Get(ctx context.Context, cid, outputPath string) error {
	node, err := cl.api.Unixfs().Get(ctx, icorepath.New(cid))
	if err != nil {
		return fmt.Errorf("failed to get file '%s': %w", cid, err)
	}

	return files.WriteTo(node, outputPath)
}

// connectToPeers connects the node to a list of IPFS bootstrap peers.
func connectToPeers(ctx context.Context, ipfs icore.CoreAPI, bootstrapNodes []string) error {
	var wg sync.WaitGroup
	peerInfos := make(map[peer.ID]*peer.AddrInfo, len(bootstrapNodes))
	for _, addrStr := range bootstrapNodes {
		addr, err := ma.NewMultiaddr(addrStr)
		if err != nil {
			return err
		}
		pii, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			return err
		}
		pi, ok := peerInfos[pii.ID]
		if !ok {
			pi = &peer.AddrInfo{ID: pii.ID}
			peerInfos[pi.ID] = pi
		}
		pi.Addrs = append(pi.Addrs, pii.Addrs...)
	}

	wg.Add(len(peerInfos))
	for _, peerInfo := range peerInfos {
		go func(peerInfo *peer.AddrInfo) {
			defer wg.Done()
			if err := ipfs.Swarm().Connect(ctx, *peerInfo); err != nil {
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

	if err = createRepo(repoPath, cfg.getKeypairSize()); err != nil {
		return nil, nil, fmt.Errorf("failed to create repo: %w", err)
	}

	repo, err := fsrepo.Open(repoPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open temp repo: %w", err)
	}

	nodeOptions := &core.BuildCfg{
		Repo:    repo,
		Online:  true,
		Routing: libp2p.DHTOption, // TODO: can set to be client for gets
	}

	node, err := core.NewNode(ctx, nodeOptions)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create node: %w", err)
	}

	api, err := coreapi.NewCoreAPI(node)
	return api, node, err
}

// createRepo creates an IPFS repository in a given directory.
func createRepo(path string, keypairSize int) error {
	cfg, err := config.Init(io.Discard, keypairSize)
	if err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
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
