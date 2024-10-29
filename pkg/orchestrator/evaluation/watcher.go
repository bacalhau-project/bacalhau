package evaluation

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
)

// WatchHandler processes evaluation events from the event store and enqueues
// them into the evaluation broker.
type WatchHandler struct {
	broker orchestrator.EvaluationBroker
}

func NewWatchHandler(broker orchestrator.EvaluationBroker) *WatchHandler {
	return &WatchHandler{
		broker: broker,
	}
}

// HandleEvent processes evaluation events and enqueues new evaluations into the broker.
// It only processes creation events since deletions are handled by the broker itself.
func (h *WatchHandler) HandleEvent(ctx context.Context, event watcher.Event) error {
	// Skip non-create operations early
	if event.Operation != watcher.OperationCreate {
		return nil
	}

	// Skip non-evaluation events
	if event.ObjectType != jobstore.EventObjectEvaluation {
		return nil
	}

	eval, ok := event.Object.(*models.Evaluation)
	if !ok {
		log.Ctx(ctx).Error().
			Str("event_type", event.ObjectType).
			Msgf("Received unexpected object type: %T", event.Object)
		return nil
	}

	if err := h.broker.Enqueue(eval); err != nil {
		log.Ctx(ctx).Error().Err(err).
			Str("evaluation_id", eval.ID).
			Str("job_id", eval.JobID).
			Msg("Failed to enqueue evaluation")
		return err
	}

	return nil
}
