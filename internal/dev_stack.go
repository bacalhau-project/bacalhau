package internal

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/internal/ipfs"
	"github.com/filecoin-project/bacalhau/internal/scheduler/libp2p"
	"github.com/phayes/freeport"
	"github.com/rs/zerolog/log"
)

type DevStackNode struct {
	ComputeNode   *ComputeNode
	RequesterNode *RequesterNode
	JsonRpcPort   int
	IpfsRepo      string
}

type DevStack struct {
	Nodes []*DevStackNode
}

func NewDevStack(
	ctx context.Context,
	count, badActors int,
) (*DevStack, error) {

	nodes := []*DevStackNode{}

	bacalhauMultiAddresses := []string{}
	devstackIpfsMultiAddresses := []string{}

	// create 3 bacalhau compute nodes
	for i := 0; i < count; i++ {
		log.Debug().Msgf(`
---------------------
  Creating Node #%d
---------------------`, i)
		libp2pPort, err := freeport.GetFreePort()
		if err != nil {
			return nil, err
		}

		libp2pScheduler, err := libp2p.NewLibp2pScheduler(ctx, libp2pPort)
		if err != nil {
			return nil, err
		}

		requesterNode, err := NewRequesterNode(ctx, libp2pScheduler)
		if err != nil {
			return nil, err
		}

		computeNode, err := NewComputeNode(ctx, libp2pScheduler, badActors > i)
		if err != nil {
			return nil, err
		}

		// at this point the requester and compute nodes are both subscribing to the scheduler events
		err = libp2pScheduler.Start()
		if err != nil {
			return nil, err
		}

		bacalhauMultiAddress := fmt.Sprintf("%s/p2p/%s", libp2pScheduler.Host.Addrs()[0].String(), libp2pScheduler.Host.ID())

		log.Debug().Msgf("bacalhau multiaddress: %s\n", bacalhauMultiAddress)

		// if we have started any bacalhau servers already, use the first one
		if len(bacalhauMultiAddresses) > 0 {
			err = libp2pScheduler.Connect(bacalhauMultiAddresses[0])
			if err != nil {
				return nil, err
			}
		}

		jsonRpcPort, err := freeport.GetFreePort()
		if err != nil {
			return nil, err
		}

		RunBacalhauJsonRpcServer(ctx, "0.0.0.0", jsonRpcPort, requesterNode)

		connectToMultiAddress := ""

		// if we have started any ipfs servers already, use the first one
		if len(devstackIpfsMultiAddresses) > 0 {
			connectToMultiAddress = devstackIpfsMultiAddresses[0]
		}

		ipfsRepo, computeNodeIpfsMultiaddresses, err := ipfs.StartBacalhauDevelopmentIpfsServer(ctx, connectToMultiAddress)
		if err != nil {
			log.Error().Err(err).Msg("Unable to start Bacalhau Dev Ipfs Server")
			return nil, err
		}

		bacalhauMultiAddresses = append(bacalhauMultiAddresses, bacalhauMultiAddress)
		devstackIpfsMultiAddresses = append(devstackIpfsMultiAddresses, computeNodeIpfsMultiaddresses[0])

		computeNode.IpfsRepo = ipfsRepo

		log.Debug().Msgf("bacalhau multiaddress: %s\n", bacalhauMultiAddress)
		log.Debug().Msgf("ipfs multiaddress: %s\n", devstackIpfsMultiAddresses)
		log.Debug().Msgf("ipfs repo: %s\n", ipfsRepo)

		devStackNode := &DevStackNode{
			ComputeNode:   computeNode,
			RequesterNode: requesterNode,
			JsonRpcPort:   jsonRpcPort,
			IpfsRepo:      ipfsRepo,
		}

		nodes = append(nodes, devStackNode)
		log.Debug().Msg("==== Complete")
	}

	stack := &DevStack{
		Nodes: nodes,
	}

	log.Debug().Msg("Finished provisioning nodes.")
	return stack, nil
}
