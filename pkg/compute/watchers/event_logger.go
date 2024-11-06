package watchers

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// ExecutionLogger handles logging of execution-related events with detailed state transition information
type ExecutionLogger struct {
	logger zerolog.Logger
}

// NewExecutionLogger creates a new ExecutionLogger instance
func NewExecutionLogger(logger zerolog.Logger) *ExecutionLogger {
	return &ExecutionLogger{
		logger: logger,
	}
}

// HandleEvent processes incoming events and logs detailed execution information
func (e *ExecutionLogger) HandleEvent(ctx context.Context, event watcher.Event) error {
	// Only handle ExecutionUpsert events
	if event.ObjectType != compute.EventObjectExecutionUpsert {
		return nil
	}

	upsert, ok := event.Object.(models.ExecutionUpsert)
	if !ok {
		e.logger.Error().
			Str("event_type", event.ObjectType).
			Msg("Failed to cast event object to ExecutionUpsert")
		return nil
	}

	// Create base log event with common fields
	logEvent := e.logger.Debug().Str("execution_id", upsert.Current.ID)

	// Add state transition information if this is an update
	if upsert.Previous != nil {
		duration := time.Duration(upsert.Current.ModifyTime - upsert.Previous.ModifyTime)
		logEvent = logEvent.
			Str("previous_state", upsert.Previous.ComputeState.StateType.String()).
			Str("current_state", upsert.Current.ComputeState.StateType.String()).
			Str("previous_desired_state", upsert.Previous.DesiredState.StateType.String()).
			Str("current_desired_state", upsert.Current.DesiredState.StateType.String()).
			Int64("state_change_duration_ms", duration.Milliseconds())

		// Determine what changed
		computeStateChanged := upsert.Previous.ComputeState.StateType != upsert.Current.ComputeState.StateType
		desiredStateChanged := upsert.Previous.DesiredState.StateType != upsert.Current.DesiredState.StateType

		// Construct appropriate message based on what changed
		switch {
		case computeStateChanged && desiredStateChanged:
			logEvent.Msgf("Execution state changed from '%s' to '%s' and desired state from '%s' to '%s'",
				upsert.Previous.ComputeState.StateType,
				upsert.Current.ComputeState.StateType,
				upsert.Previous.DesiredState.StateType,
				upsert.Current.DesiredState.StateType)
		case computeStateChanged:
			logEvent.Msgf("Execution state changed from '%s' to '%s'",
				upsert.Previous.ComputeState.StateType,
				upsert.Current.ComputeState.StateType)
		case desiredStateChanged:
			logEvent.Msgf("Execution desired state changed from '%s' to '%s'",
				upsert.Previous.DesiredState.StateType,
				upsert.Current.DesiredState.StateType)
		default:
			logEvent.Msg("Execution updated with no state changes")
		}
	} else {
		// This is a new execution
		logEvent = logEvent.
			Str("initial_state", upsert.Current.ComputeState.StateType.String()).
			Str("desired_state", upsert.Current.DesiredState.StateType.String())

		logEvent.Msg("New execution created")
	}

	// Log associated events if any
	if len(upsert.Events) > 0 {
		eventLogger := e.logger.Trace().
			Str("execution_id", upsert.Current.ID)

		for _, ev := range upsert.Events {
			eventLogger = eventLogger.
				Str("event_topic", string(ev.Topic)).
				Time("event_time", ev.Timestamp)

			if ev.Message != "" {
				eventLogger = eventLogger.Str("event_message", ev.Message)
			}
		}

		eventLogger.Msgf("Execution generated %d new events", len(upsert.Events))
	}

	return nil
}
