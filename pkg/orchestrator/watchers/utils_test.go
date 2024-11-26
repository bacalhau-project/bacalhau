package watchers

import (
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

// setupNewExecution creates an upsert for a new execution with no previous state
func setupNewExecution(
	desiredState models.ExecutionDesiredStateType,
	computeState models.ExecutionStateType,
	events ...*models.Event,
) models.ExecutionUpsert {
	execution := mock.Execution()
	execution.ComputeState = models.NewExecutionState(computeState)
	execution.DesiredState = models.NewExecutionDesiredState(desiredState)

	return models.ExecutionUpsert{
		Previous: nil,
		Current:  execution,
		Events:   events,
	}
}

// setupStateTransition creates an upsert for an execution state transition
func setupStateTransition(
	prevDesiredState models.ExecutionDesiredStateType,
	prevComputeState models.ExecutionStateType,
	newDesiredState models.ExecutionDesiredStateType,
	newComputeState models.ExecutionStateType,
	events ...*models.Event,
) models.ExecutionUpsert {
	previous := mock.Execution()
	previous.ComputeState = models.NewExecutionState(prevComputeState)
	previous.DesiredState = models.NewExecutionDesiredState(prevDesiredState)

	current := mock.Execution()
	current.ID = previous.ID // Ensure same execution
	current.JobID = previous.JobID
	current.NodeID = previous.NodeID
	current.ComputeState = models.NewExecutionState(newComputeState)
	current.DesiredState = models.NewExecutionDesiredState(newDesiredState)

	return models.ExecutionUpsert{
		Previous: previous,
		Current:  current,
		Events:   events,
	}
}

// createExecutionEvent is a helper to create watcher.Event from an ExecutionUpsert
func createExecutionEvent(upsert models.ExecutionUpsert) watcher.Event {
	return watcher.Event{
		Object: upsert,
	}
}
