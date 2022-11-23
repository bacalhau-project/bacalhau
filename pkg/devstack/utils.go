package devstack

import (
	"github.com/filecoin-project/bacalhau/pkg/ipfs"

	"github.com/filecoin-project/bacalhau/pkg/node"
)

func ToIPFSClients(nodes []*node.Node) []*ipfs.Client {
	res := []*ipfs.Client{}
	for _, n := range nodes {
		res = append(res, n.IPFSClient)
	}

	return res
}
