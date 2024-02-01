package system

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/pbnjay/memory"
	"github.com/ricochet2200/go-disk-usage/du"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity/system/gpu"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type PhysicalCapacityProvider struct {
	gpuCapacityProviders []capacity.Provider
}

func NewPhysicalCapacityProvider() *PhysicalCapacityProvider {
	return &PhysicalCapacityProvider{
		gpuCapacityProviders: []capacity.Provider{
			gpu.NewNvidiaGPUProvider(),
			gpu.NewAMDGPUProvider(),
			gpu.NewIntelGPUProvider(),
		},
	}
}

func (p *PhysicalCapacityProvider) GetAvailableCapacity(ctx context.Context) (models.Resources, error) {
	totalCapacity, err := p.GetTotalCapacity(ctx)
	if err != nil {
		return totalCapacity, err
	}

	return models.Resources{
		CPU:    totalCapacity.CPU * 0.8,
		Memory: totalCapacity.Memory * 80 / 100,
		Disk:   totalCapacity.Disk * 80 / 100,
		GPU:    totalCapacity.GPU,
		GPUs:   totalCapacity.GPUs,
	}, nil
}

func (p *PhysicalCapacityProvider) GetTotalCapacity(ctx context.Context) (models.Resources, error) {
	diskSpace, err := getFreeDiskSpace(config.GetStoragePath())
	if err != nil {
		return models.Resources{}, err
	}

	// the total resources we have
	resources := models.Resources{
		CPU:    float64(runtime.NumCPU()),
		Memory: memory.TotalMemory(),
		Disk:   diskSpace,
	}

	for _, gpuProvider := range p.gpuCapacityProviders {
		gpuCapacity, err := gpuProvider.GetAvailableCapacity(ctx)
		if err != nil {
			// we won't error here since some hosts may have the tool installed
			// but in a misconfigured state e.g. their drivers are missing, the
			// smi can't communicate with the drivers, etc. instead we provide a
			// warning, show the args to the command we tried and its response.
			// motivation: https://expanso.atlassian.net/browse/GDAY-90
			log.Ctx(ctx).Debug().Msgf(
				"Cannot inspect %s so they will not be used: %s",
				strings.Join(gpuProvider.ResourceTypes(), " or "),
				err.Error(),
			)
			continue
		}

		resources = *resources.Add(gpuCapacity)
	}

	return resources, nil
}

// ResourceTypes implements capacity.Provider.
func (p *PhysicalCapacityProvider) ResourceTypes() []string {
	ownTypes := []string{"CPU", "Memory", "Disk"}
	for _, gpuProvider := range p.gpuCapacityProviders {
		ownTypes = append(ownTypes, gpuProvider.ResourceTypes()...)
	}
	return ownTypes
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

// compile-time check that the provider implements the interface
var _ capacity.Provider = (*PhysicalCapacityProvider)(nil)
