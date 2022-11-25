//go:build unit || !integration

package apicopy

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
)

// how many bytes more does ipfs report the file than the actual content?
const IpfsMetadataSize uint64 = 8

func TestGetVolumeSize(t *testing.T) {
	ctx := context.Background()

	for _, testString := range []string{
		"hello from test volume size",
		"hello world",
	} {
		t.Run(testString, func(t *testing.T) {
			cm := system.NewCleanupManager()
			t.Cleanup(cm.Cleanup)

			node, err := ipfs.NewLocalNode(ctx, cm, []string{})
			require.NoError(t, err)

			apiAddresses, err := node.APIAddresses()
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(apiAddresses), 1)

			storage, err := NewStorage(cm, apiAddresses[0])
			require.NoError(t, err)

			cid, err := ipfs.AddTextToNodes(ctx, []byte(testString), storage.IPFSClient)
			require.NoError(t, err)

			result, err := storage.GetVolumeSize(ctx, model.StorageSpec{
				StorageSource: model.StorageSourceIPFS,
				CID:           cid,
				Path:          "/",
			})

			require.NoError(t, err)
			require.Equal(t, uint64(len(testString))+IpfsMetadataSize, result)
		})
	}
}
