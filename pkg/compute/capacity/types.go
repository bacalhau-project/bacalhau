package capacity

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

type Tracker interface {
	IsWithinLimits(ctx context.Context, usage model.ResourceUsageData) bool
	AddIfHasCapacity(ctx context.Context, usage model.ResourceUsageData) bool
	AvailableCapacity(ctx context.Context) model.ResourceUsageData
	Remove(ctx context.Context, usage model.ResourceUsageData)
}

type UsageCalculator interface {
	Calculate(ctx context.Context, job model.Job, parsedUsage model.ResourceUsageData) (model.ResourceUsageData, error)
}

type Provider interface {
	AvailableCapacity(ctx context.Context) (model.ResourceUsageData, error)
}
