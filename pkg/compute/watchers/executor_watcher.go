package watchers

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/rs/zerolog/log"
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
	upsert, ok := event.Object.(models.ExecutionUpsert)
	if !ok {
		return fmt.Errorf("failed to cast event object to models.ExecutionUpsert. Found type %T", event.Object)
	}

	execution := upsert.Current
	logger := log.Ctx(ctx).With().
		Str("executionID", execution.ID).
		Str("state", execution.ComputeState.StateType.String()).
		Logger()

	var err error
	switch execution.ComputeState.StateType {
	case models.ExecutionStateNew:
		if err = h.bidder.RunBidding(ctx, execution); err != nil {
			compute.ExecutionBiddingErrors.Add(ctx, 1)
			logger.Error().
				Err(err).
				Msg("failed to run bidding")
		}
	case models.ExecutionStateBidAccepted:
		err = h.executor.Run(ctx, execution)
		if err != nil {
			compute.ExecutionRunErrors.Add(ctx, 1)
			logger.Error().
				Err(err).
				Msg("failed to run execution")
		}
	case models.ExecutionStateCancelled:
 case models.ExecutionStateCancelled:
   err = h.executor.Cancel(ctx, execution)
   if err != nil {
+    logger.Error().Err(err).Msg("failed to cancel execution")
     compute.ExecutionCancelErrors.Add(ctx, 1)
   }
	default:
		// No action needed for other states
		return nil
	}

	if err != nil {
		return bacerrors.Wrap(err, "failed to handle execution state %s", execution.ComputeState.StateType)
	}
	return nil
}
