package disk

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/models"
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
	ctx context.Context, execution *models.Execution, parsedUsage models.Resources) (*models.Resources, error) {
	requirements := &models.Resources{}

	var totalDiskRequirements uint64 = 0
	for _, input := range execution.Job.Task().InputSources {
		strg, err := c.storages.Get(ctx, input.Source.Type)
		if err != nil {
			return nil, err
		}
		volumeSize, err := strg.GetVolumeSize(ctx, execution, *input)
		if err != nil {
			return nil, bacerrors.Wrap(err, "error getting job disk space requirements")
		}
		totalDiskRequirements += volumeSize
	}

	// update the job requirements disk space with what we calculated
	requirements.Disk = totalDiskRequirements

	return requirements, nil
}

var _ capacity.UsageCalculator = (*DiskUsageCalculator)(nil)
