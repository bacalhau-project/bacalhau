package ipfs

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/hashicorp/go-multierror"
	icore "github.com/ipfs/boxo/coreiface"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/pkg/errors"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/system"

	"github.com/rs/zerolog/log"

	"github.com/ipfs/kubo/commands"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreapi"
	"github.com/ipfs/kubo/core/corehttp"
	"github.com/ipfs/kubo/core/node/libp2p"
	"github.com/ipfs/kubo/plugin/loader"
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
	// PvtIpfsFolderPerm is what permissions we give to a private ipfs repo
	PvtIpfsFolderPerm = 0755
)

// Node is a wrapper around an in-process IPFS node that can be used to
// interact with the IPFS network without requiring an `ipfs` binary.
type Node struct {
	api      icore.CoreAPI
	ipfsNode *core.IpfsNode
	cfg      types.IpfsConfig

	// RepoPath is the path to the ipfs node's data repository.
	RepoPath string

	// APIPort is the port that the node's ipfs API is listening on.
	APIPort int

	apiAddresses []string
}

// NewNodeWithConfig creates a new in-process IPFS node with the given configuration.
func NewNodeWithConfig(ctx context.Context, cm *system.CleanupManager, cfg types.IpfsConfig) (*Node, error) {
	var err error
	pluginOnce.Do(func() {
		err = loadPlugins(cm)
	})
	if err != nil {
		return nil, err
	}

	api, ipfsNode, repoPath, err := createNode(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create ipfs node: %w", err)
	}
	defer func() {
		if err != nil {
			_ = ipfsNode.Close()
		}
	}()

	// TODO if cfg.PrivateInternal is true do we actually want to connect to peers?
	if err = connectToPeers(ctx, api, ipfsNode, cfg.GetSwarmAddresses()); err != nil {
		log.Ctx(ctx).Error().Msgf("ipfs node failed to connect to peers: %s", err)
	}

	apiAddresses, err := serveAPI(cm, ipfsNode, repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to serve API: %w", err)
	}

	var apiPort int
	if len(apiAddresses) > 0 {
		apiPort, err = getTCPPort(apiAddresses[0])
		if err != nil {
			return nil, fmt.Errorf("failed to parse api port: %w", err)
		}
	}

	n := Node{
		api:      api,
		ipfsNode: ipfsNode,
		cfg:      cfg,
		RepoPath: repoPath,
		APIPort:  apiPort,
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

// SwarmAddresses returns the node's swarm addresses.
func (n *Node) SwarmAddresses() ([]string, error) {
	addresses, err := n.api.Swarm().ListenAddrs(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to get repo config: %w", err)
	}

	var res []string
	for _, addr := range addresses {
		res = append(res, fmt.Sprintf("%s/p2p/%s", addr, n.ID()))
	}

	return res, nil
}

// LogDetails logs connection details for the node's swarm and API servers.
func (n *Node) LogDetails() {
	id := n.ID()

	swarmAddrs, err := n.SwarmAddresses()
	if err != nil {
		log.Debug().Msgf("error fetching swarm addresses: %s", err)
	} else {
		log.Trace().Str("id", id).Strs("addresses", swarmAddrs).Msg("IPFS node listening for swarm")
	}
	log.Trace().Str("id", id).Strs("addresses", n.apiAddresses).Msg("IPFS node listening for API")
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
			if err := n.ipfsNode.Repo.Close(); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("failed to close repo: %w", err))
			}
		}
	}

	// delete repo if user didn't specify a repo path.
	if n.cfg.ServePath == "" {
		if err := os.RemoveAll(n.RepoPath); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("failed to clean up repo directory: %w", err))
		}
	}
	return errs.ErrorOrNil()
}

// createNode spawns a new IPFS node using a temporary repo path.
func createNode(ctx context.Context, cfg types.IpfsConfig) (icore.CoreAPI, *core.IpfsNode,
	string, error) {
	// generate an IPFS configuration
	ipfsCfg, err := buildIPFSConfig(cfg)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to build IPFS config")
	}

	// we need to check if the user wants a custom repo path and if there is already a repo present there.
	repoPath := cfg.ServePath
	if repoPath == "" {
		// user doesn't want a custom repo path, we will make temporary repo that's removed when the node calls Close()
		repoPath, err = os.MkdirTemp("", "ipfs-tmp")
		if err != nil {
			return nil, nil, "", fmt.Errorf("failed to make temporary IPFS repo: %w", err)
		}
		if err := fsrepo.Init(repoPath, ipfsCfg); err != nil {
			return nil, nil, "", fmt.Errorf("failed to initalize IPFS repo at %s: %w", repoPath, err)
		}
	} else {
		// user wants deterministic repo path, check if one is already present
		if _, err := os.Stat(repoPath); err != nil {
			if os.IsNotExist(err) {
				if err := fsrepo.Init(repoPath, ipfsCfg); err != nil {
					return nil, nil, "", fmt.Errorf("failed to initalize IPFS repo at %s: %w", repoPath, err)
				}
			} else {
				return nil, nil, "", err
			}
		}
	}

	// If we have a swarm key, copy it into the repo
	if cfg.SwarmKeyPath != "" {
		destinationPath := filepath.Join(repoPath, "swarm.key")
		err = copyFile(cfg.SwarmKeyPath, destinationPath)
		if err != nil {
			return nil, nil, "", fmt.Errorf("failed to copy swarm key: %w", err)
		} else {
			log.Ctx(ctx).Debug().Str("Path", destinationPath).Msg("Copied IPFS private swarm key")
		}
	}

	repo, err := fsrepo.Open(repoPath)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to open repo: %w", err)
	}

	// if the user provided an IPFS repo then we need to copy the peerID and private key
	// from the existing repo to the configuration we built. This will allow the embedded IPFS node to keep
	// its identity across restarts.
	if cfg.ServePath != "" {
		repoCfg, err := repo.Config()
		if err != nil {
			return nil, nil, "", err
		}
		ipfsCfg.Identity.PeerID = repoCfg.Identity.PeerID
		ipfsCfg.Identity.PrivKey = repoCfg.Identity.PrivKey
	}

	// write the configuration we built to the IPFS repo.
	if err := repo.SetConfig(ipfsCfg); err != nil {
		return nil, nil, "", err
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
func serveAPI(cm *system.CleanupManager, node *core.IpfsNode, repoPath string) ([]string, error) {
	cfg, err := node.Repo.Config()
	if err != nil {
		return nil, fmt.Errorf("failed to get repo config: %w", err)
	}

	var listeners []manet.Listener
	for _, addr := range cfg.Addresses.API {
		maddr, err := ma.NewMultiaddr(addr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse multiaddr: %w", err)
		}

		listener, err := manet.Listen(maddr)
		if err != nil {
			return nil, fmt.Errorf("failed to listen on api multiaddr: %w", err)
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
		corehttp.WebUIOption,
		corehttp.CommandsOption(cmdContext),
	}

	var addresses []string
	for _, listener := range listeners {
		addresses = append(addresses, listener.Multiaddr().String())
		log.Debug().Stringer("Address", listener.Multiaddr()).Msg("IPFS listening")
		// NOTE: this is not critical, but we should log for debugging
		go func(listener manet.Listener) {
			err := corehttp.Serve(node, manet.NetListener(listener), opts...)
			if err != nil && !errors.Is(err, net.ErrClosed) {
				log.Warn().Stringer("IPFSNode", node.Identity).Err(err).Msg("failed to serve IPFS API")
			}
		}(listener)
	}

	return addresses, nil
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
				log.Ctx(ctx).Debug().Err(err).Msgf("failed to connect to ipfs peer %s, skipping", peerInfo.ID)
			}
		}(peerInfo)
	}

	wg.Wait()
	return anyErr
}

// loadPlugins initializes and injects the standard set of ipfs plugins.
func loadPlugins(cm *system.CleanupManager) error {
	// We use a temporary folder for the plugin loader so that when it
	// does the dynamic plugin loading (which cannot be turned off) it
	// does not break when it finds a binary in a local ./plugins folder
	repoDir, err := os.MkdirTemp("", "")
	if err != nil {
		return err
	}

	plugins, err := loader.NewPluginLoader(repoDir)
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
	cm.RegisterCallback(func() error {
		plugins.Close()
		return os.RemoveAll(repoDir)
	})
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
