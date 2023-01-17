package ethblock

import (
	"context"
	"math/big"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockEthFetcher struct {
	blocks  map[int64]*types.Block
	index   int
}

func newMockEthFetcher() *mockEthFetcher {
	return &mockEthFetcher{
		blocks:  map[int64]*types.Block{
			1: types.NewBlockWithHeader(&types.Header{
				Number: new(big.Int).SetInt64(1),
			}),
			2: types.NewBlockWithHeader(&types.Header{
				Number: new(big.Int).SetInt64(2),
			}),
		},
	}
}

var _ EthFetcher = (*mockEthFetcher)(nil)

func (f *mockEthFetcher) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	return f.blocks[number.Int64()], nil
}

func newTestDriver(t *testing.T) *StorageProvider {
	driver, err := NewStorage(newMockEthFetcher(), t.TempDir())
	require.NoError(t, err)
	return driver
}

func TestIsInstalled(t *testing.T) {
	driver := newTestDriver(t)
	isInstalled, err := driver.IsInstalled(context.Background())
	require.NoError(t, err)
	assert.True(t, isInstalled)
}

func TestHasStorageLocally(t *testing.T) {
	tests := []struct{
		blocksContents string
		logsContents string
	}{
		{
			blocksContents: "",
			logsContents: "",
		},
		{
			blocksContents: `
[
    ""
]
			`,
		},
	}

	t.Run("no storage", func(t *testing.T) {
		driver := newTestDriver(t)
		has, err := driver.HasStorageLocally(context.Background(), model.StorageSpec{
			BlockRange: &model.BlockRange{
				Start: new(big.Int).SetInt64(1),
				End: new(big.Int).SetInt64(2),
			},
		})
		require.NoError(t, err)
		assert.False(t, has)
	})

	t.Run("", func(t *testing.T) {
		driver := newTestDriver(t)
		has, err := driver.HasStorageLocally(context.Background(), model.StorageSpec{
			BlockRange: &model.BlockRange{
				Start: new(big.Int).SetInt64(1),
				End: new(big.Int).SetInt64(2),
			},
		})
		require.NoError(t, err)
		assert.False(t, has)
	})
}
