//go:build unit || !integration

package ipfs

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs/ipfstesting"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// how many bytes more does ipfs report the file than the actual content?
const IpfsMetadataSize uint64 = 8

func getIpfsStorage(t *testing.T) *StorageProvider {
	storage, err := NewStorage(ipfstesting.NewFakeIPFSNode())
	require.NoError(t, err)

	return storage
}

func TestGetVolumeSize(t *testing.T) {
	t.Skip("The objective of this test is not understood")
	ctx := context.Background()
	config.SetVolumeSizeRequestTimeout(time.Second * 3)

	for _, testString := range []string{
		"hello from test volume size",
		"hello world",
	} {
		t.Run(testString, func(t *testing.T) {
			storage := getIpfsStorage(t)

			cid, err := ipfstesting.AddTextToNodes(ctx, []byte(testString), storage.ipfsClient)
			require.NoError(t, err)

			result, err := storage.GetVolumeSize(ctx, models.InputSource{
				Source: &models.SpecConfig{
					Type: models.StorageSourceIPFS,
					Params: Source{
						CID: cid,
					}.ToMap(),
				},
				Target: "/",
			})

			require.NoError(t, err)
			require.Equal(t, uint64(len(testString))+IpfsMetadataSize, result, "expected %d actual %d", uint64(len(testString))+IpfsMetadataSize, result)
		})
	}
}

func TestPrepareStorageRespectsTimeouts(t *testing.T) {
	t.Skip("The objective of this test is not understood")
	for _, testDuration := range []time.Duration{
		//0, // Disable test -- timeouts aren't respected when getting cached files
		time.Minute,
	} {
		t.Run(fmt.Sprint(testDuration), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testDuration)
			defer cancel()
			storage := getIpfsStorage(t)

			cid, err := ipfstesting.AddTextToNodes(ctx, []byte("testString"), storage.ipfsClient)
			require.NoError(t, err)

			_, err = storage.PrepareStorage(ctx, t.TempDir(), models.InputSource{
				Source: &models.SpecConfig{
					Type: models.StorageSourceIPFS,
					Params: Source{
						CID: cid,
					}.ToMap(),
				},
				Target: "/",
			})
			require.Equal(t, testDuration == 0, err != nil)
		})
	}
}

func TestGetVolumeSizeRespectsTimeout(t *testing.T) {
	t.Skip("The objective of this test is not understood")
	for _, testDuration := range []time.Duration{
		// 0, // Disable test -- timeouts aren't respected when getting cached files
		time.Minute,
	} {
		t.Run(fmt.Sprint(testDuration), func(t *testing.T) {
			ctx := context.Background()
			storage := getIpfsStorage(t)

			cid, err := ipfstesting.AddTextToNodes(ctx, []byte("testString"), storage.ipfsClient)
			require.NoError(t, err)

			config.SetVolumeSizeRequestTimeout(testDuration)
			_, err = storage.GetVolumeSize(ctx, models.InputSource{
				Source: &models.SpecConfig{
					Type: models.StorageSourceIPFS,
					Params: Source{
						CID: cid,
					}.ToMap(),
				},
				Target: "/",
			})
			require.Equal(t, testDuration == 0, err != nil)
		})
	}
}
