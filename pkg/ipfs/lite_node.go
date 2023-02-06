package ipfs

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/hashicorp/go-multierror"
	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/kubo/repo"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog/log"

	"github.com/ipfs/kubo/config"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreapi"
	"github.com/libp2p/go-libp2p/core/peer"
)

type LiteNodeParams struct {
	// PeerAddrs is a list of peers to connect to.
	PeerAddrs []string
}

// LiteNode is a wrapper around an in-process low power IPFS node with limited functionality and nilRepo
// that should only be used to download content from IPFS.
type LiteNode struct {
	api      icore.CoreAPI
	ipfsNode *core.IpfsNode
}

type LiteClient interface {
	Get(ctx context.Context, cid, outputPath string) error
}

// NewLiteNode creates a new IPFS lite
func NewLiteNode(ctx context.Context, params LiteNodeParams) (*LiteNode, error) {
	var err error
	var ipfsRepo repo.Repo
	defer func() {
		if err != nil && ipfsRepo != nil {
			_ = ipfsRepo.Close()
		}
	}()

	ipfsRepo, err = createLiteRepo(params.PeerAddrs)
	if err != nil {
		return nil, fmt.Errorf("failed to create repo: %w", err)
	}
	ipfsNode, err := core.NewNode(ctx, &core.BuildCfg{
		Repo:   ipfsRepo,
		Online: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create node: %w", err)
	}

	api, err := coreapi.NewCoreAPI(ipfsNode)
	if err != nil {
		return nil, fmt.Errorf("failed to create coreapi: %w", err)
	}

	err = connectToPeers(ctx, ipfsNode, params.PeerAddrs)
	if err != nil {
		return nil, err
	}
	log.Trace().Msgf("IPFS lite node created with ID: %s", ipfsNode.Identity)

	return &LiteNode{
		api:      api,
		ipfsNode: ipfsNode,
	}, nil
}

func createLiteRepo(peers []string) (repo.Repo, error) {
	cfg := &config.Config{}
	profiles := []string{"lowpower", "randomports"}

	for _, profile := range profiles {
		transformer, ok := config.Profiles[profile]
		if !ok {
			return nil, fmt.Errorf("invalid configuration profile: %s", profile)
		}
		if err := transformer.Transform(cfg); err != nil { //nolint: govet
			return nil, err
		}
	}

	priv, pub, err := crypto.GenerateKeyPairWithReader(crypto.RSA, defaultKeypairSize, rand.Reader)
	if err != nil {
		return nil, err
	}

	pid, err := peer.IDFromPublicKey(pub)
	if err != nil {
		return nil, err
	}

	privkeyb, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return nil, err
	}

	bootstrapPeers := config.DefaultBootstrapAddresses
	for _, peerAddr := range peers {
		if peerAddr != "" {
			bootstrapPeers = append(bootstrapPeers, peerAddr)
		}
	}
	cfg.Bootstrap = bootstrapPeers
	cfg.Identity.PeerID = pid.String()
	cfg.Identity.PrivKey = base64.StdEncoding.EncodeToString(privkeyb)

	return &repo.Mock{
		D: dsync.MutexWrap(ds.NewNullDatastore()),
		C: *cfg,
	}, nil
}

// ID returns the node's ipfs ID.
func (n *LiteNode) ID() string {
	return n.ipfsNode.Identity.String()
}

// SwarmAddresses returns the node's swarm addresses.
func (n *LiteNode) SwarmAddresses() ([]string, error) {
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

// Client returns an API client for interacting with the node.
func (n *LiteNode) Client() LiteClient {
	return NewClient(n.api)
}

func (n *LiteNode) Close() error {
	log.Debug().Msgf("Closing IPFS lite node %s", n.ID())
	var errs *multierror.Error
	if n.ipfsNode != nil {
		errs = multierror.Append(errs, n.ipfsNode.Close())

		if n.ipfsNode.Repo != nil {
			if err := n.ipfsNode.Repo.Close(); err != nil { //nolint:govet
				errs = multierror.Append(errs, fmt.Errorf("failed to close repo: %w", err))
			}
		}
	}
	return errs.ErrorOrNil()
}
