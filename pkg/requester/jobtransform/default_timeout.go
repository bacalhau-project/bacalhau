package jobtransform

import (
	"context"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

// Sets a default timeout value if one is not passed
func NewTimeoutApplier(defaultTimeout time.Duration) Transformer {
	return func(ctx context.Context, job *model.Job) (modified bool, err error) {
		if job.Spec.GetTimeout() <= 0 {
			job.Spec.Timeout = int64(defaultTimeout.Seconds())
			return true, nil
		}
		return
	}
}
