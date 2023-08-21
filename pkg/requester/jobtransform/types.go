package jobtransform

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// Transformer is a function that can be used to modify a job before it is migrated to a new version.
type Transformer func(context.Context, *model.Job) (modified bool, err error)

// PostTransformer is a function that can be used to modify a job after it has been migrated to a new version.
type PostTransformer func(context.Context, *models.Job) (modified bool, err error)
