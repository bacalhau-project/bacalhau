package ipfs

import (
	"context"
	"testing"

	ipfs_http "github.com/filecoin-project/bacalhau/pkg/ipfs/http"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func TestIpfsHTTPClient(t *testing.T) {
	ctx := context.Background()
	stack, cm := SetupTest(t, 2)
	defer TeardownTest(stack, cm)

	fileCid, err := stack.AddTextToNodes(1, []byte(`hello world`))
	assert.NoError(t, err)

	// test the basic connection and that we can list the IPFS node addresses
	ipfsMultiAddress := stack.Nodes[0].IpfsNode.APIAddress()
	api, err := ipfs_http.NewIPFSHTTPClient(ipfsMultiAddress)
	assert.NoError(t, err)

	addrs, err := api.GetLocalAddrs(ctx)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(addrs), 1)

	assertNodeHasCid := func(cid string, nodeIndex int, expectedResult bool) {
		api, err := ipfs_http.NewIPFSHTTPClient(
			stack.Nodes[nodeIndex].IpfsNode.APIAddress())
		assert.NoError(t, err)

		result, err := api.HasCidLocally(ctx, cid)
		assert.NoError(t, err)

		assert.Equal(t, expectedResult, result)
	}

	assertNodeHasCid(fileCid, 0, true)
	assertNodeHasCid(fileCid, 1, false)
}
