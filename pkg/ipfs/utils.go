package ipfs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/filecoin-project/bacalhau/pkg/storage/util"
	"github.com/ipfs/kubo/core"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

func AddFileToNodes(ctx context.Context, filePath string, clients ...Client) (string, error) {
	var res string
	for i, client := range clients {
		cid, err := client.Put(ctx, filePath)
		if err != nil {
			return "", errors.Wrap(err, fmt.Sprintf("error adding %q to node %d", filePath, i))
		}

		log.Ctx(ctx).Debug().Msgf("Added CID %q to IPFS node %q", cid, client.APIAddress())
		res = strings.TrimSpace(cid)
	}

	return res, nil
}

func AddTextToNodes(ctx context.Context, fileContent []byte, clients ...Client) (string, error) {
	tempDir, err := os.MkdirTemp("", "bacalhau-test")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tempDir)

	testFilePath := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFilePath, fileContent, util.OS_USER_RW|util.OS_ALL_R)
	if err != nil {
		return "", err
	}

	return AddFileToNodes(ctx, testFilePath, clients...)
}

// connectToPeers connects the node to a list of IPFS bootstrap peers.
func connectToPeers(ctx context.Context, node *core.IpfsNode, peerAddrs []string) error {
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
			if err := node.PeerHost.Connect(ctx, *peerInfo); err != nil {
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
