package devstack

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

type DevStackIPFS struct {
	IPFSClients    []ipfs.Client
	CleanupManager *system.CleanupManager
}

// NewDevStackIPFS creates a devstack but with only IPFS servers connected to each other
func NewDevStackIPFS(ctx context.Context, cm *system.CleanupManager, count int) (*DevStackIPFS, error) {
	var clients []ipfs.Client
	for i := 0; i < count; i++ {
		log.Ctx(ctx).Debug().Msgf(`Creating Node #%d`, i)

		//////////////////////////////////////
		// IPFS
		//////////////////////////////////////
		var err error
		var ipfsSwarmAddrs []string
		if i > 0 {
			ipfsSwarmAddrs, err = clients[0].SwarmAddresses(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get ipfs swarm addresses: %w", err)
			}
		}

		ipfsNode, err := ipfs.NewLocalNode(ctx, cm, ipfsSwarmAddrs)
		if err != nil {
			return nil, fmt.Errorf("failed to create ipfs node: %w", err)
		}

		clients = append(clients, ipfsNode.Client())
	}

	stack := &DevStackIPFS{
		IPFSClients:    clients,
		CleanupManager: cm,
	}

	return stack, nil
}
