package ipfs_test

import (
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/ipfs"
)

func TestRoundTrip(t *testing.T) {
	expectedCid, err := cid.Decode("QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG")
	require.NoError(t, err)
	expectedSpec := ipfs.IPFSStorageSpec{
		CID: expectedCid,
	}

	spec, err := expectedSpec.AsSpec("name", "mount")
	require.NoError(t, err)

	require.NotEmpty(t, spec.SchemaData)
	require.NotEmpty(t, spec.Params)
	require.True(t, ipfs.Schema.Cid().Equals(spec.Schema))

	t.Log(string(spec.SchemaData))
	t.Log(string(spec.Params))

	actualSpec, err := ipfs.Decode(spec)
	require.NoError(t, err)

	assert.True(t, expectedSpec.CID.Equals(actualSpec.CID))
	assert.True(t, actualSpec.CID.Equals(expectedCid))
}
