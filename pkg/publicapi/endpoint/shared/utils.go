package shared

import (
	"github.com/ricochet2200/go-disk-usage/du"
)

type HealthInfo struct {
	DiskFreeSpace FreeSpace `json:"FreeSpace"`
}

type FreeSpace struct {
	IPFSMount MountStatus `json:"IPFSMount"`
	TMP       MountStatus `json:"tmp"`
	ROOT      MountStatus `json:"root"`
}

// Creating structure for DiskStatus
type MountStatus struct {
	All  uint64 `json:"All"`
	Used uint64 `json:"Used"`
	Free uint64 `json:"Free"`
}

func GenerateHealthData() HealthInfo {
	var healthInfo HealthInfo

	// Generating all, free, used amounts for each - in case these are different mounts, they'll have different
	// All and Free values, if they're all on the same machine, then those values should be the same
	// If "All" is 0, that means the directory does not exist
	healthInfo.DiskFreeSpace.IPFSMount = MountUsage("/data/ipfs")
	healthInfo.DiskFreeSpace.ROOT = MountUsage("/")
	healthInfo.DiskFreeSpace.TMP = MountUsage("/tmp")

	return healthInfo
}

// Function to get disk usage of path/disk
func MountUsage(path string) (disk MountStatus) {
	usage := du.NewDiskUsage(path)
	if usage == nil {
		return
	}
	disk.All = usage.Size()
	disk.Free = usage.Free()
	disk.Used = usage.Used()
	return
}
