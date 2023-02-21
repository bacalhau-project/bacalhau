package jobtransform

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/pkg/errors"
)

func NewExecutionPlanner(provider storage.StorageProvider) Transformer {
	return func(ctx context.Context, j *model.Job) (modified bool, err error) {
		executionPlan, err := job.GenerateExecutionPlan(ctx, j.Spec, provider)
		if err != nil {
			return false, errors.Wrap(err, "error generating execution plan")
		}
		j.Spec.ExecutionPlan = executionPlan
		return true, nil
	}
}
