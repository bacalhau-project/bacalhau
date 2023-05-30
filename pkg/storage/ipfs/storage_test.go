//go:build unit || !integration

package ipfs

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	spec_ipfs "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

// how many bytes more does ipfs report the file than the actual content?
const IpfsMetadataSize uint64 = 8

func getIpfsStorage(t *testing.T) *StorageProvider {
	ctx := context.Background()
	cm := system.NewCleanupManager()
	t.Cleanup(func() {
		cm.Cleanup(context.Background())
	})

	node, err := ipfs.NewLocalNode(ctx, cm, []string{})
	require.NoError(t, err)

	storage, err := NewStorage(cm, node.Client())
	require.NoError(t, err)

	return storage
}

func TestGetVolumeSize(t *testing.T) {
	ctx := context.Background()

	for _, testString := range []string{
		"hello from test volume size",
		"hello world",
	} {
		t.Run(testString, func(t *testing.T) {
			storage := getIpfsStorage(t)

			cidstr, err := ipfs.AddTextToNodes(ctx, []byte(testString), storage.ipfsClient)
			require.NoError(t, err)

			c, err := cid.Decode(cidstr)
			require.NoError(t, err)

			ipfsspec, err := (&spec_ipfs.IPFSStorageSpec{CID: c}).AsSpec("TODO", "/")
			require.NoError(t, err)

			result, err := storage.GetVolumeSize(ctx, ipfsspec)

			require.NoError(t, err)
			require.Equal(t, uint64(len(testString))+IpfsMetadataSize, result)
		})
	}
}

func TestPrepareStorageRespectsTimeouts(t *testing.T) {
	for _, testDuration := range []time.Duration{
		// 0, // Disable test -- timeouts aren't respected when getting cached files
		time.Minute,
	} {
		t.Run(fmt.Sprint(testDuration), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testDuration)
			defer cancel()
			storage := getIpfsStorage(t)

			cidstr, err := ipfs.AddTextToNodes(ctx, []byte("testString"), storage.ipfsClient)
			require.NoError(t, err)

			c, err := cid.Decode(cidstr)
			require.NoError(t, err)

			ipfsspec, err := (&spec_ipfs.IPFSStorageSpec{CID: c}).AsSpec("TODO", "/")
			require.NoError(t, err)

			_, err = storage.PrepareStorage(ctx, ipfsspec)
			require.Equal(t, testDuration == 0, err != nil)
		})
	}
}

func TestGetVolumeSizeRespectsTimeout(t *testing.T) {
	for _, testDuration := range []time.Duration{
		// 0, // Disable test -- timeouts aren't respected when getting cached files
		time.Minute,
	} {
		t.Run(fmt.Sprint(testDuration), func(t *testing.T) {
			ctx := context.Background()
			storage := getIpfsStorage(t)

			cidstr, err := ipfs.AddTextToNodes(ctx, []byte("testString"), storage.ipfsClient)
			require.NoError(t, err)

			c, err := cid.Decode(cidstr)
			require.NoError(t, err)

			ipfsspec, err := (&spec_ipfs.IPFSStorageSpec{CID: c}).AsSpec("TODO", "/")
			require.NoError(t, err)

			ctx = config.SetVolumeSizeRequestTimeout(ctx, testDuration)
			_, err = storage.GetVolumeSize(ctx, ipfsspec)
			require.Equal(t, testDuration == 0, err != nil)
		})
	}
}
