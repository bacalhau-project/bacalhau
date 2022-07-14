package capacitymanager

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestItemList(t *testing.T) {
	list := NewItemList()
	list.Add(CapacityManagerItem{ID: "1", Requirements: ResourceUsageData{CPU: 0.1, Memory: 100, Disk: 100}})
	item := list.Get("1")
	require.NotNil(t, item)
	require.Equal(t, item.ID, "1")
	require.Equal(t, float64(0.1), item.Requirements.CPU)
	require.Equal(t, uint64(100), item.Requirements.Memory)
	require.Equal(t, uint64(100), item.Requirements.Disk)
	require.Equal(t, 1, list.Count())
	list.Remove("1")
	require.Equal(t, 0, list.Count())
	require.Nil(t, list.Get("1"))
}
