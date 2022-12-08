package system

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/compute/capacity"
	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/pbnjay/memory"
	"github.com/ricochet2200/go-disk-usage/du"
)

// NvidiaCLI is the path to the Nvidia helper binary
const NvidiaCLI = "nvidia-container-cli"

type PhysicalCapacityProvider struct {
}

func NewPhysicalCapacityProvider() *PhysicalCapacityProvider {
	return &PhysicalCapacityProvider{}
}

func (p *PhysicalCapacityProvider) AvailableCapacity(ctx context.Context) (model.ResourceUsageData, error) {
	diskSpace, err := getFreeDiskSpace(config.GetStoragePath())
	if err != nil {
		return model.ResourceUsageData{}, err
	}
	gpus, err := numSystemGPUs()
	if err != nil {
		return model.ResourceUsageData{}, err
	}

	// the actual resources we have
	return model.ResourceUsageData{
		CPU:    float64(runtime.NumCPU()) * 0.8,
		Memory: memory.TotalMemory() * 80 / 100,
		Disk:   diskSpace * 80 / 100,
		GPU:    gpus,
	}, nil
}

// get free disk space for storage path
// returns bytes
func getFreeDiskSpace(path string) (uint64, error) {
	usage := du.NewDiskUsage(path)
	if usage == nil {
		return 0, fmt.Errorf("getFreeDiskSpace: unable to get disk space for path %s", path)
	}
	return usage.Free(), nil
}

// numSystemGPUs wraps nvidia-container-cli to get the number of GPUs
func numSystemGPUs() (uint64, error) {
	nvidiaPath, err := exec.LookPath(NvidiaCLI)
	if err != nil {
		// If the NVIDIA CLI is not installed, we can't know the number of GPUs, assume zero
		if (err.(*exec.Error)).Unwrap() == exec.ErrNotFound {
			return 0, nil
		}
		return 0, err
	}
	args := []string{
		"info",
		"--csv",
	}
	cmd := exec.Command(nvidiaPath, args...)
	resp, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	// Parse output of nvidia-container-cli command
	lines := strings.Split(string(resp), "\n")
	deviceInfoFlag := false
	numDevices := uint64(0)
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.HasPrefix(line, "Device Index") {
			deviceInfoFlag = true
			continue
		}
		if deviceInfoFlag {
			numDevices += 1
		}
	}

	fmt.Println(numDevices)
	return numDevices, nil
}

// compile-time check that the provider implements the interface
var _ capacity.Provider = (*PhysicalCapacityProvider)(nil)
