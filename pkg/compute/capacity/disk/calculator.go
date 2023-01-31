package disk

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/model"
)

type DiskUsageCalculatorParams struct {
	Executors executor.ExecutorProvider
}

type DiskUsageCalculator struct {
	executors executor.ExecutorProvider
}

func NewDiskUsageCalculator(params DiskUsageCalculatorParams) *DiskUsageCalculator {
	return &DiskUsageCalculator{
		executors: params.Executors,
	}
}

func (c *DiskUsageCalculator) Calculate(
	ctx context.Context, job model.Job, parsedUsage model.ResourceUsageData) (model.ResourceUsageData, error) {
	requirements := model.ResourceUsageData{}

	e, err := c.executors.Get(ctx, job.Spec.Engine)
	if err != nil {
		return model.ResourceUsageData{}, fmt.Errorf("error getting job disk space requirements: %w", err)
	}

	var totalDiskRequirements uint64 = 0

	for _, input := range job.Spec.Inputs {
		volumeSize, err := e.GetVolumeSize(ctx, input)
		if err != nil {
			return model.ResourceUsageData{}, fmt.Errorf("error getting job disk space requirements: %w", err)
		}
		totalDiskRequirements += volumeSize
	}

	// TODO: think about the fact that each shard might be different sizes
	//  this is probably good enough for now
	totalShards := job.Spec.ExecutionPlan.TotalShards
	if totalShards == 0 {
		totalShards = 1
	}
	// update the job requirements disk space with what we calculated
	requirements.Disk = totalDiskRequirements / uint64(totalShards)

	return requirements, nil
}
