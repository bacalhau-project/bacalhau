package compute

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
)

type ExecutionStateHandler struct {
	executor Executor
}

func NewExecutionStateHandler(executor Executor) *ExecutionStateHandler {
	return &ExecutionStateHandler{
		executor: executor,
	}
}

func (h *ExecutionStateHandler) HandleEvent(ctx context.Context, event watcher.Event) error {
	// TODO: filter out old events, or make sure we don't get them during node startup
	localExecution, ok := event.Object.(store.LocalExecutionState)
	if !ok {
		return fmt.Errorf("failed to cast event object to LocalExecutionState. Found type %T", event.Object)
	}

	switch localExecution.State {
	case store.ExecutionStateBidAccepted:
		return h.executor.Run(ctx, localExecution)
	case store.ExecutionStateCancelled:
		return h.executor.Cancel(ctx, localExecution)
	default:
	}

	return nil
}
