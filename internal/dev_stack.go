package internal

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/internal/ipfs"
	"github.com/filecoin-project/bacalhau/internal/logger"
	"github.com/filecoin-project/bacalhau/internal/scheduler/libp2p"
	"github.com/opentracing/opentracing-go/log"
	"github.com/phayes/freeport"
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
	ipfsMultiAddresses := []string{}

	// create 3 bacalhau compute nodes
	for i := 0; i < count; i++ {
		logger.Debug("---------------------")
		logger.Infof("	Creating Node #%d", i)
		logger.Debug("---------------------")
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

		logger.Debugf("bacalhau multiaddress: %s\n", bacalhauMultiAddress)

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
		if len(ipfsMultiAddresses) > 0 {
			connectToMultiAddress = ipfsMultiAddresses[0]
		}

		ipfsRepo, ipfsMultiaddress, err := ipfs.StartBacalhauDevelopmentIpfsServer(ctx, connectToMultiAddress)
		if err != nil {
			log.Error(err)
			return nil, err
		}

		bacalhauMultiAddresses = append(bacalhauMultiAddresses, bacalhauMultiAddress)
		ipfsMultiAddresses = append(ipfsMultiAddresses, ipfsMultiaddress)

		computeNode.IpfsRepo = ipfsRepo
		computeNode.IpfsConnectMultiAddress = ipfsMultiaddress

		logger.Debugf("bacalhau multiaddress: %s\n", bacalhauMultiAddress)
		logger.Debugf("ipfs multiaddress: %s\n", ipfsMultiaddress)
		logger.Debugf("ipfs repo: %s\n", ipfsRepo)

		devStackNode := &DevStackNode{
			ComputeNode:   computeNode,
			RequesterNode: requesterNode,
			JsonRpcPort:   jsonRpcPort,
			IpfsRepo:      ipfsRepo,
		}

		nodes = append(nodes, devStackNode)
		logger.Debug("==== Complete")
	}

	stack := &DevStack{
		Nodes: nodes,
	}

	logger.Debug("Finished provisioning nodes.")
	return stack, nil
}
