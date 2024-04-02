package shared

import (
	"github.com/bacalhau-project/bacalhau/pkg/types"
	"github.com/ricochet2200/go-disk-usage/du"
)

func GenerateHealthData() types.HealthInfo {
	var healthInfo types.HealthInfo

	// Generating all, free, used amounts for each - in case these are different mounts, they'll have different
	// All and Free values, if they're all on the same machine, then those values should be the same
	// If "All" is 0, that means the directory does not exist
	healthInfo.DiskFreeSpace.IPFSMount = MountUsage("/data/ipfs")
	healthInfo.DiskFreeSpace.ROOT = MountUsage("/")
	healthInfo.DiskFreeSpace.TMP = MountUsage("/tmp")

	return healthInfo
}

// Function to get disk usage of path/disk
func MountUsage(path string) (disk types.MountStatus) {
	usage := du.NewDiskUsage(path)
	if usage == nil {
		return
	}
	disk.All = usage.Size()
	disk.Free = usage.Free()
	disk.Used = usage.Used()
	return
}
