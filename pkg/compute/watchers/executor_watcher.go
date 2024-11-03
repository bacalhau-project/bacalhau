package watchers

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type ExecutionUpsertHandler struct {
	executor compute.Executor
	bidder   compute.Bidder
}

func NewExecutionUpsertHandler(executor compute.Executor, bidder compute.Bidder) *ExecutionUpsertHandler {
	return &ExecutionUpsertHandler{
		executor: executor,
		bidder:   bidder,
	}
}

func (h *ExecutionUpsertHandler) HandleEvent(ctx context.Context, event watcher.Event) error {
	// TODO: filter out old events, or make sure we don't get them during node startup
	upsert, ok := event.Object.(store.ExecutionUpsert)
	if !ok {
		return fmt.Errorf("failed to cast event object to store.ExecutionUpsert. Found type %T", event.Object)
	}

	execution := upsert.Current
	switch execution.ComputeState.StateType {
	case models.ExecutionStateNew:
		return h.bidder.RunBidding(ctx, execution)
	case models.ExecutionStateBidAccepted:
		return h.executor.Run(ctx, execution)
	case models.ExecutionStateCancelled:
		return h.executor.Cancel(ctx, execution)
	default:
	}

	return nil
}
