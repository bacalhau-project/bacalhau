package publicapi

import (
	"os/exec"
	"strconv"

	"github.com/bacalhau-project/bacalhau/pkg/types"
	"github.com/ricochet2200/go-disk-usage/du"
	"github.com/rs/zerolog/log"
)

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

// use "-1" as count for just last line
func TailFile(count int, path string) ([]byte, error) {
	c := exec.Command("tail", strconv.Itoa(count), path) //nolint:gosec // subprocess not at risk
	output, err := c.Output()
	if err != nil {
		log.Warn().Msgf("Could not find file at %s", path)
		return nil, err
	}
	return output, nil
}
