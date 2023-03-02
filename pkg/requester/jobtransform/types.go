package jobtransform

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type Transformer func(context.Context, *model.Job) (modified bool, err error)
