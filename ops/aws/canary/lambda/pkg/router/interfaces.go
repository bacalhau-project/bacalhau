package router

import (
	"context"
)

type Handler func(ctx context.Context) error
