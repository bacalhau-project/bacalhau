package transformer

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// GenericTransformer is an interface that can be used to modify an object in place.
type GenericTransformer[T any] interface {
	Transform(context.Context, T) error
}

// JobTransformer is an interface that can be used to modify a job in place,
// such as setting default values or migrating.
type JobTransformer interface {
	GenericTransformer[*models.Job]
}

// ResultTransformer is an interface that can be used to modify a result in place.
type ResultTransformer interface {
	GenericTransformer[*models.SpecConfig]
}

// JobFn is a function that implements JobTransformer transform interface.
type JobFn func(context.Context, *models.Job) error

func (fn JobFn) Transform(ctx context.Context, job *models.Job) error {
	return fn(ctx, job)
}

// ResultFn is a function that implements ResultTransformer transform interface.
type ResultFn func(context.Context, *models.SpecConfig) error

func (fn ResultFn) Transform(ctx context.Context, result *models.SpecConfig) error {
	return fn(ctx, result)
}

// compile time check that JobFn and ResultFn implement the interfaces
var _ JobTransformer = JobFn(nil)
var _ ResultTransformer = ResultFn(nil)
