package analytics

import (
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// resource represents the resource usage of a job.
type resource struct {
	CPUUnits    float64   `json:"cpu_units,omitempty"`
	MemoryBytes uint64    `json:"memory_bytes,omitempty"`
	DiskBytes   uint64    `json:"disk_bytes,omitempty"`
	GPUCount    uint64    `json:"gpu_count,omitempty"`
	GPUTypes    []gpuInfo `json:"gpu_types,omitempty"`
}

// gpuInfo represents information about a GPU device.
type gpuInfo struct {
	Name   string `json:"name,omitempty"`
	Vendor string `json:"vendor,omitempty"`
}

// newResourceFromConfig creates a new resource from a models.ResourcesConfig.
// If the config cannot be parsed, it returns a zero-valued resource.
func newResourceFromConfig(config *models.ResourcesConfig) resource {
	if config == nil {
		return resource{}
	}

	taskResources, err := config.ToResources()
	if err != nil {
		return resource{}
	}

	gpuTypes := make([]gpuInfo, len(taskResources.GPUs))
	for i, gpu := range taskResources.GPUs {
		gpuTypes[i] = gpuInfo{
			Name:   gpu.Name,
			Vendor: string(gpu.Vendor),
		}
	}

	return resource{
		CPUUnits:    taskResources.CPU,
		MemoryBytes: taskResources.Memory,
		DiskBytes:   taskResources.Disk,
		GPUCount:    taskResources.GPU,
		GPUTypes:    gpuTypes,
	}
}
