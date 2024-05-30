package router

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

type Handler func(ctx context.Context, cfg types.BacalhauConfig) error
