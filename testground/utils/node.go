package utils

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/localdb/inmemory"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/libp2p"
	"github.com/multiformats/go-multiaddr"
	"github.com/phayes/freeport"
)

type NodeInfo struct {
	Libp2pAddrs []string
	IPFSAddrs   []string
}

var _ fmt.Stringer = NodeInfo{}

func (ni NodeInfo) String() string {
	return fmt.Sprintf("{libp2p: %v, ipfs: %v}", ni.Libp2pAddrs, ni.IPFSAddrs)
}

func (ni NodeInfo) GetLibp2pAddrs() ([]multiaddr.Multiaddr, error) {
	addrs := []multiaddr.Multiaddr{}
	for _, addr := range ni.Libp2pAddrs {
		maddr, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			return nil, err
		}
		addrs = append(addrs, maddr)
	}
	return addrs, nil
}

func CreateAndStartNode(ctx context.Context,
	cm *system.CleanupManager,
	bootstrapNode *NodeInfo) (*node.Node, error) {
	newNode, err := CreateNode(ctx, cm, bootstrapNode)
	if err != nil {
		return nil, err
	}
	err = newNode.Start(ctx)
	if err != nil {
		return nil, err
	}
	return newNode, nil
}

func CreateNode(ctx context.Context,
	cm *system.CleanupManager,
	bootstrapNode *NodeInfo) (*node.Node, error) {
	ipfsNode, err := ipfs.NewLocalNode(ctx, cm, bootstrapNode.IPFSAddrs)
	if err != nil {
		return nil, fmt.Errorf("failed to create ipfs node: %w", err)
	}

	ipfsClient, err := ipfsNode.Client()
	if err != nil {
		return nil, fmt.Errorf("failed to create ipfs client: %w", err)
	}

	peers, err := bootstrapNode.GetLibp2pAddrs()
	if err != nil {
		return nil, err
	}
	transport, err := libp2p.NewTransportFromOptions(ctx, cm, peers)
	if err != nil {
		return nil, err
	}

	apiPort, err := freeport.GetFreePort()
	if err != nil {
		return nil, err
	}

	metricsPort, err := freeport.GetFreePort()
	if err != nil {
		return nil, err
	}

	datastore, err := inmemory.NewInMemoryDatastore()
	if err != nil {
		return nil, err
	}

	nodeConfig := node.NodeConfig{
		IPFSClient:          ipfsClient,
		CleanupManager:      cm,
		LocalDB:             datastore,
		Transport:           transport,
		HostAddress:         "0.0.0.0",
		APIPort:             apiPort,
		MetricsPort:         metricsPort,
		ComputeNodeConfig:   node.NewComputeConfigWithDefaults(),
		RequesterNodeConfig: requesternode.RequesterNodeConfig{},
	}

	// Start transport layer
	err = transport.Start(ctx)
	if err != nil {
		return nil, err
	}

	return node.NewStandardNode(ctx, nodeConfig)
}

func GetNodeInfo(ctx context.Context, node *node.Node) (*NodeInfo, error) {
	ipfsSwarmAddrs, err := node.IPFSClient.SwarmAddresses(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get ipfs swarm addresses: %w", err)
	}

	nodeLibp2pAddrs, err := node.Transport.HostAddrs()
	if err != nil {
		return nil, fmt.Errorf("failed to get node addresses: %w", err)
	}

	libp2pAddrs := []string{}
	for _, addr := range nodeLibp2pAddrs {
		libp2pAddrs = append(libp2pAddrs, addr.String())
	}

	nodeInfo := &NodeInfo{
		Libp2pAddrs: libp2pAddrs,
		IPFSAddrs:   ipfsSwarmAddrs,
	}
	return nodeInfo, nil
}
