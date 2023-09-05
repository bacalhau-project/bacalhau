package transformer

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// Job is an interface that can be used to modify a job in place,
// such as setting default values or migrating.
type Job interface {
	Transform(context.Context, *models.Job) error
}

// JobFn is a function that implements Job transform interface.
type JobFn func(context.Context, *models.Job) error

func (fn JobFn) Transform(ctx context.Context, job *models.Job) error {
	return fn(ctx, job)
}

// compile time check that JobFn implements Job
var _ Job = JobFn(nil)
