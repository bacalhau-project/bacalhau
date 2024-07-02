package planner

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
)

// EventEmitter is a planner implementation that emits events based on the job state.
type EventEmitter struct {
	id           string
	eventEmitter orchestrator.EventEmitter
}

// EventEmitterParams holds the parameters for creating a new EventEmitter.
type EventEmitterParams struct {
	ID           string
	EventEmitter orchestrator.EventEmitter
}

// NewEventEmitter creates a new instance of EventEmitter.
func NewEventEmitter(params EventEmitterParams) *EventEmitter {
	return &EventEmitter{
		id:           params.ID,
		eventEmitter: params.EventEmitter,
	}
}

// Process updates the state of the executions in the plan according to the scheduler's desired state.
func (s *EventEmitter) Process(ctx context.Context, plan *models.Plan) error {
	var eventName models.JobEventType
	switch plan.DesiredJobState {
	case models.JobStateTypeCompleted:
		eventName = models.JobEventCompleted
	case models.JobStateTypeFailed:
		eventName = models.JobEventError
	default:
	}
	if !eventName.IsUndefined() {
		s.eventEmitter.EmitEventSilently(ctx, models.JobEvent{
			SourceNodeID: s.id,
			JobID:        plan.Job.ID,
			Status:       plan.Event.Message,
			EventName:    eventName,
			EventTime:    plan.Event.Timestamp,
		})
	}
	return nil
}

// compile-time check whether the EventEmitter implements the Planner interface.
var _ orchestrator.Planner = (*EventEmitter)(nil)
