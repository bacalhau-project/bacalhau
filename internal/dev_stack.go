package internal

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/internal/ipfs"
	"github.com/phayes/freeport"
)

type DevStackNode struct {
	Node        *ComputeNode
	JsonRpcPort int
	IpfsRepo    string
}

type DevStack struct {
	Nodes []*DevStackNode
}

func NewDevStack(
	ctx context.Context,
	count int,
) (*DevStack, error) {

	nodes := []*DevStackNode{}

	bacalhauMultiAddresses := []string{}
	ipfsMultiAddresses := []string{}
	ipfsRepos := []string{}

	// create 3 bacalhau compute nodes
	for i := 0; i < count; i++ {
		computePort, err := freeport.GetFreePort()
		if err != nil {
			return nil, err
		}
		node, err := NewComputeNode(ctx, computePort)
		if err != nil {
			return nil, err
		}

		bacalhauMultiAddress := fmt.Sprintf("%s/p2p/%s", node.Host.Addrs()[0].String(), node.Host.ID())

		fmt.Printf("bacalhau multiaddress: %s\n", bacalhauMultiAddress)

		// if we have started any bacalhau servers already, use the first one
		if len(bacalhauMultiAddresses) > 0 {
			err = node.Connect(bacalhauMultiAddresses[0])
			if err != nil {
				return nil, err
			}
		}

		jsonRpcPort, err := freeport.GetFreePort()
		if err != nil {
			return nil, err
		}

		go RunBacalhauJsonRpcServer(ctx, "0.0.0.0", jsonRpcPort, nil)

		connectToMultiAddress := ""

		// if we have started any ipfs servers already, use the first one
		if len(ipfsMultiAddresses) > 0 {
			connectToMultiAddress = ipfsMultiAddresses[0]
		}

		ipfsRepo, ipfsMultiaddress, err := ipfs.StartBacalhauDevelopmentIpfsServer(ctx, connectToMultiAddress)
		if err != nil {
			return nil, err
		}

		bacalhauMultiAddresses = append(bacalhauMultiAddresses, bacalhauMultiAddress)
		ipfsMultiAddresses = append(ipfsMultiAddresses, ipfsMultiaddress)
		ipfsRepos = append(ipfsRepos, ipfsRepo)
		node.IpfsRepo = ipfsRepo
		node.IpfsConnectMultiAddress = ipfsMultiaddress

		fmt.Printf("bacalhau multiaddress: %s\n", bacalhauMultiAddress)
		fmt.Printf("ipfs multiaddress: %s\n", ipfsMultiaddress)
		fmt.Printf("ipfs repo: %s\n", ipfsRepo)

		devStackNode := &DevStackNode{
			Node:        node,
			JsonRpcPort: jsonRpcPort,
			IpfsRepo:    ipfsRepo,
		}

		nodes = append(nodes, devStackNode)
	}

	stack := &DevStack{
		Nodes: nodes,
	}

	return stack, nil
}
