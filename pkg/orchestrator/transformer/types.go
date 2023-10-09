package transformer

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// JobTransformer is an interface that can be used to modify a job in place,
// such as setting default values or migrating.
type JobTransformer interface {
	Transform(context.Context, *models.Job) error
}

// JobFn is a function that implements JobTransformer transform interface.
type JobFn func(context.Context, *models.Job) error

func (fn JobFn) Transform(ctx context.Context, job *models.Job) error {
	return fn(ctx, job)
}

// compile time check that JobFn implements JobTransformer
var _ JobTransformer = JobFn(nil)
