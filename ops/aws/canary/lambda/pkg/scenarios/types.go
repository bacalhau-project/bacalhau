package scenarios

import (
	"context"
)

type Event struct {
	Action string `json:"action"`
}

type Handler func(ctx context.Context) error
