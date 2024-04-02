//go:build unit || !integration

package capacity

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/stretchr/testify/require"
)

func TestAllocatesGPUs(t *testing.T) {
	tracker := NewLocalTracker(LocalTrackerParams{MaxCapacity: models.Resources{
		GPU: 2,
		GPUs: []models.GPU{
			{Index: 0, Name: "Lancer 2X", Vendor: models.GPUVendorNvidia, Memory: 100},
			{Index: 1, Name: "Berdly 1.0", Vendor: models.GPUVendorAMDATI, Memory: 100},
		},
	}})

	added := tracker.AddIfHasCapacity(context.Background(), models.Resources{GPU: 1})
	require.NotNil(t, added)
	require.Equal(t, uint64(1), added.GPU)

	avail := tracker.GetAvailableCapacity(context.Background())
	require.Equal(t, uint64(1), avail.GPU)
	require.Len(t, avail.GPUs, 1)
	require.NotEqual(t, avail.GPUs[0], added.GPUs[0])
}

func TestDoesntAllocateGPUs(t *testing.T) {
	tracker := NewLocalTracker(LocalTrackerParams{MaxCapacity: models.Resources{
		GPU: 2,
		GPUs: []models.GPU{
			{Index: 0, Name: "Lancer 2X", Vendor: models.GPUVendorNvidia, Memory: 100},
			{Index: 1, Name: "Berdly 1.0", Vendor: models.GPUVendorAMDATI, Memory: 100},
		},
	}})

	added := tracker.AddIfHasCapacity(context.Background(), models.Resources{GPU: 4})
	require.Nil(t, added)

	avail := tracker.GetAvailableCapacity(context.Background())
	require.Equal(t, uint64(2), avail.GPU)
	require.Len(t, avail.GPUs, 2)
	require.Equal(t, avail, tracker.maxCapacity)
}
