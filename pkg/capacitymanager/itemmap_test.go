package capacitymanager

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestItemMap(t *testing.T) {
	itemMap := NewItemMap()
	itemMap.Add(CapacityManagerItem{ID: "1", Requirements: ResourceUsageData{CPU: 0.1, Memory: 100, Disk: 100}})
	item := itemMap.Get("1")
	require.NotNil(t, item)
	require.Equal(t, item.ID, "1")
	require.Equal(t, float64(0.1), item.Requirements.CPU)
	require.Equal(t, uint64(100), item.Requirements.Memory)
	require.Equal(t, uint64(100), item.Requirements.Disk)
	require.Equal(t, 1, itemMap.Count())
	itemMap.Remove("1")
	require.Equal(t, 0, itemMap.Count())
	require.Nil(t, itemMap.Get("1"))
}
