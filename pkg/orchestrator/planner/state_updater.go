package planner

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
)

// StateUpdater is responsible for updating the state of executions and jobs in the plan.
// It makes sense to have this as the first planner in the chain, so that the desired state
// is updated before any other planner try to execute the plan, such as forwarding an execution
// to a compute node.
type StateUpdater struct {
	store jobstore.Store
}

// NewStateUpdater creates a new instance of StateUpdater with the specified jobstore.Store.
func NewStateUpdater(store jobstore.Store) *StateUpdater {
	return &StateUpdater{
		store: store,
	}
}

// Process updates the state of the executions in the plan according to the scheduler's desired state.
func (s *StateUpdater) Process(ctx context.Context, plan *models.Plan) error {
	// TODO: evaluate the need for partial failure handling instead of failing and retrying
	//  the whole evaluation/plan.

	// Create new executions
	for _, exec := range plan.NewExecutions {
		err := s.store.CreateExecution(ctx, *exec, plan.Event)
		if err != nil {
			return err
		}
	}

	// Update existing executions
	for _, u := range plan.UpdatedExecutions {
		err := s.store.UpdateExecution(ctx, jobstore.UpdateExecutionRequest{
			ExecutionID: u.Execution.ID,
			NewValues: models.Execution{
				DesiredState: models.State[models.ExecutionDesiredStateType]{
					StateType: u.DesiredState,
					Message:   u.Event.Message,
				},
			},
			Condition: jobstore.UpdateExecutionCondition{
				ExpectedRevision: u.Execution.Revision,
			},
			Event: u.Event,
		})
		if err != nil {
			return err
		}
	}

	// Update job state if necessary
	if !plan.DesiredJobState.IsUndefined() {
		err := s.store.UpdateJobState(ctx, jobstore.UpdateJobStateRequest{
			JobID:    plan.Job.ID,
			NewState: plan.DesiredJobState,
			Event:    plan.Event,
			Condition: jobstore.UpdateJobCondition{
				ExpectedRevision: plan.Job.Revision,
			},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// compile-time check whether the StateUpdater implements the Planner interface.
var _ orchestrator.Planner = (*StateUpdater)(nil)
