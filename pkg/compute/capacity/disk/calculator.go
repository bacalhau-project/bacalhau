package disk

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

type DiskUsageCalculatorParams struct {
	Storages storage.StorageProvider
}

type DiskUsageCalculator struct {
	storages storage.StorageProvider
}

func NewDiskUsageCalculator(params DiskUsageCalculatorParams) *DiskUsageCalculator {
	return &DiskUsageCalculator{
		storages: params.Storages,
	}
}

func (c *DiskUsageCalculator) Calculate(
	ctx context.Context, job models.Job, parsedUsage models.Resources) (models.Resources, error) {
	requirements := models.Resources{}

	var totalDiskRequirements uint64 = 0
	for _, input := range job.Spec.Inputs {
		strg, err := c.storages.Get(ctx, input.StorageSource)
		if err != nil {
			return models.Resources{}, err
		}
		volumeSize, err := strg.GetVolumeSize(ctx, input)
		if err != nil {
			return models.Resources{}, fmt.Errorf("error getting job disk space requirements: %w", err)
		}
		totalDiskRequirements += volumeSize
	}

	// update the job requirements disk space with what we calculated
	requirements.Disk = totalDiskRequirements

	return requirements, nil
}
