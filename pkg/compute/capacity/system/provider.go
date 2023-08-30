package system

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strconv"

	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/pbnjay/memory"
	"github.com/ricochet2200/go-disk-usage/du"
)

type GPU struct {
	// Self-reported index of the device in the system
	Index uint64
	// Model name of the GPU e.g. Tesla T4
	Name string
	// Total GPU memory in mebibytes (MiB)
	Memory uint64
}

type PhysicalCapacityProvider struct {
}

func NewPhysicalCapacityProvider() *PhysicalCapacityProvider {
	return &PhysicalCapacityProvider{}
}

func (p *PhysicalCapacityProvider) GetAvailableCapacity(ctx context.Context) (models.Resources, error) {
	diskSpace, err := getFreeDiskSpace(config.GetStoragePath())
	if err != nil {
		return models.Resources{}, err
	}
	gpus, err := numSystemGPUs()
	if err != nil {
		return models.Resources{}, err
	}

	// the actual resources we have
	return models.Resources{
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

func numSystemGPUs() (uint64, error) {
	gpus, err := GetSystemGPUs()
	return uint64(len(gpus)), err
}

// nvidiaCLI is the path to the Nvidia helper binary
const nvidiaCLI = "nvidia-smi"

// nvidiaCLIArgs is the args we pass the nvidiaCLI
var nvidiaCLIArgs = []string{
	"--query-gpu=index,gpu_name,memory.total",
	"--format=csv,noheader,nounits",
}

func GetSystemGPUs() ([]GPU, error) {
	nvidiaPath, err := exec.LookPath(nvidiaCLI)
	if err != nil {
		// If the NVIDIA CLI is not installed, we can't know the number of GPUs.
		// It is not an error to assume zero.
		return []GPU{}, nil
	}
	cmd := exec.Command(nvidiaPath, nvidiaCLIArgs...)
	resp, err := cmd.Output()
	if err != nil {
		return []GPU{}, err
	}

	return parseNvidiaCliOutput(bytes.NewReader(resp))
}

func parseNvidiaCliOutput(resp io.Reader) ([]GPU, error) {
	reader := csv.NewReader(resp)
	reader.TrimLeadingSpace = true

	records, err := reader.ReadAll()
	if err != nil {
		return []GPU{}, err
	}

	gpus := make([]GPU, len(records))
	for index, record := range records {
		gpus[index].Index, err = strconv.ParseUint(record[0], 10, 64)
		if err != nil {
			return gpus, err
		}

		gpus[index].Name = record[1]
		gpus[index].Memory, err = strconv.ParseUint(record[2], 10, 64)
		if err != nil {
			return gpus, err
		}
	}

	return gpus, nil
}

// compile-time check that the provider implements the interface
var _ capacity.Provider = (*PhysicalCapacityProvider)(nil)
