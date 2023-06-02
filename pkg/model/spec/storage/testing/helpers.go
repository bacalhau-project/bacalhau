package storagetesting

import (
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	spec_filecoin "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/filecoin"
	spec_ipfs "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/ipfs"
	spec_s3 "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/s3"
	spec_url "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/url"
)

func MakeIpfsStorageSpec(t testing.TB, name, mount, cidstr string) spec.Storage {
	c, err := cid.Decode(cidstr)
	require.NoError(t, err)
	out, err := (&spec_ipfs.IPFSStorageSpec{CID: c}).AsSpec(name, mount)
	require.NoError(t, err)
	return out
}

func MakeFilecoinStorageSpec(t testing.TB, name, mount, cidstr, dealStr string) spec.Storage {
	cc, err := cid.Decode(cidstr)
	require.NoError(t, err)
	dc, err := cid.Decode(dealStr)
	require.NoError(t, err)
	out, err := (&spec_filecoin.FilecoinStorageSpec{CID: cc, Deal: dc}).AsSpec(name, mount)
	require.NoError(t, err)
	return out
}

func MakeS3StorageSpec(t testing.TB, name, mount string, s3spec *spec_s3.S3StorageSpec) spec.Storage {
	out, err := s3spec.AsSpec(name, mount)
	require.NoError(t, err)
	return out

}

func MakeUrlStorageSpec(t testing.TB, name, mount, url string) spec.Storage {
	out, err := (&spec_url.URLStorageSpec{URL: url}).
		AsSpec(name, mount)
	require.NoError(t, err)
	return out
}
