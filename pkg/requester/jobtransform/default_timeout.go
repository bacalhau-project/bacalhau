package jobtransform

import (
	"context"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

// Sets a default timeout value if one is not passed or below an acceptable value
func NewTimeoutApplier(minTimeout, defaultTimeout time.Duration) Transformer {
	return func(ctx context.Context, job *model.Job) (modified bool, err error) {
		if job.Spec.GetTimeout() <= minTimeout {
			job.Spec.Timeout = defaultTimeout.Seconds()
			return true, nil
		}
		return
	}
}
