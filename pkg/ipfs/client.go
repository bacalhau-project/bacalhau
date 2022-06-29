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

// For loading ipfs plugins once per process:
var pluginOnce sync.Once

// A list of IPFS nodes we can use as bootstrap peers:
var bootstrapNodes = []string{
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
}

// Client is a wrapper around an in-process IPFS node that can be used to
// interact with the IPFS network without requiring an `ipfs` binary.
type Client struct {
	api    icore.CoreAPI
	node   *core.IpfsNode
	cancel context.CancelFunc
}

var nBitsForKeypair = 2048

// NewClient creates a new IPFS client.
func NewClient(cm *system.CleanupManager) (*Client, error) {
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

	api, node, err := createNode(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create ipfs node: %w", err)
	}

	// Connecting to bootstrap peers can be done asynchronously:
	go func() {
		if err := connectToPeers(ctx, api); err != nil {
			log.Error().Msgf("ipfs client failed to connect to peers: %s", err)
		}
	}()

	return &Client{
		api:    api,
		node:   node,
		cancel: cancel,
	}, nil
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
func connectToPeers(ctx context.Context, ipfs icore.CoreAPI) error {
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
func createNode(ctx context.Context) (icore.CoreAPI, *core.IpfsNode, error) {
	repoPath, err := createTempRepo()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create temp repo: %w", err)
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

// createTempRepo creates an IPFS repository in some ephemeral directory.
func createTempRepo() (string, error) {
	repoPath, err := os.MkdirTemp("", "ipfs-tmp")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	cfg, err := config.Init(io.Discard, nBitsForKeypair)
	if err != nil {
		return "", err
	}

	err = fsrepo.Init(repoPath, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to init ipfs repo: %w", err)
	}

	return repoPath, nil
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
