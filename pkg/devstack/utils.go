package devstack

import (
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/node"
)

func ToIPFSClients(nodes []*node.Node) []ipfs.Client {
	res := make([]ipfs.Client, 0, len(nodes))
	for _, n := range nodes {
		res = append(res, n.IPFSClient)
	}

	return res
}
