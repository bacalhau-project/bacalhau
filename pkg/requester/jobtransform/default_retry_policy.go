package jobtransform

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/transformer"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

func NewDefaultRetryPolicyApplier() Transformer {
	return func(ctx context.Context, job *model.Job) (modified bool, err error) {
		if job.Spec.SchedulingTimeout == 0 {
			job.Spec.SchedulingTimeout = transformer.DefaultSchedulingTimeout
		}

		if job.Spec.BaseRetryDelay == 0 {
			job.Spec.BaseRetryDelay = transformer.DefaultBaseRetryDelay
		}

		if job.Spec.MaximumRetryDelay == 0 {
			job.Spec.MaximumRetryDelay = transformer.DefaultMaximumRetryDelay
		}

		if job.Spec.RetryDelayGrowthFactor < 1.0 {
			job.Spec.RetryDelayGrowthFactor = transformer.DefaultRetryDelayGrowthFactor
		}

		return
	}
}
