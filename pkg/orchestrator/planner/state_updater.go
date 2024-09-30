package planner

import (
	"context"
	"fmt"

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
//
//nolint:gocyclo
func (s *StateUpdater) Process(ctx context.Context, plan *models.Plan) error {
	// If there are no new or updated executions
	// and the job state is not being updated, there is nothing to do.
	if len(plan.NewExecutions) == 0 &&
		len(plan.UpdatedExecutions) == 0 &&
		len(plan.NewEvaluations) == 0 &&
		plan.DesiredJobState.IsUndefined() {
		return nil
	}

	txContext, err := s.store.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			_ = txContext.Rollback()
		}
	}()

	// Create new executions
	for _, exec := range plan.NewExecutions {
		if err = s.store.CreateExecution(txContext, *exec); err != nil {
			return err
		}
	}

	// Update existing executions
	for _, u := range plan.UpdatedExecutions {
		if err = s.store.UpdateExecution(txContext, jobstore.UpdateExecutionRequest{
			ExecutionID: u.Execution.ID,
			NewValues: models.Execution{
				DesiredState: models.State[models.ExecutionDesiredStateType]{
					StateType: u.DesiredState,
					Message:   u.Event.Message,
					Details:   u.Event.Details,
				},
			},
			Condition: jobstore.UpdateExecutionCondition{
				ExpectedRevision: u.Execution.Revision,
			},
		}); err != nil {
			return err
		}
	}

	// Update job state if necessary
	if !plan.DesiredJobState.IsUndefined() {
		if err = s.store.UpdateJobState(txContext, jobstore.UpdateJobStateRequest{
			JobID:    plan.Job.ID,
			NewState: plan.DesiredJobState,
			Message:  plan.UpdateMessage,
			Condition: jobstore.UpdateJobCondition{
				ExpectedRevision: plan.Job.Revision,
			},
		}); err != nil {
			return err
		}
	}

	// Create follow-up evaluations, if any
	for _, eval := range plan.NewEvaluations {
		if err = s.store.CreateEvaluation(txContext, *eval); err != nil {
			return err
		}
	}

	// Add history events
	if len(plan.JobEvents) > 0 {
		if err = s.store.AddJobHistory(txContext, plan.Job.ID, plan.JobEvents...); err != nil {
			return err
		}
	}
	if len(plan.ExecutionEvents) > 0 {
		for executionID, events := range plan.ExecutionEvents {
			if err = s.store.AddExecutionHistory(txContext, plan.Job.ID, executionID, events...); err != nil {
				return err
			}
		}
	}

	return txContext.Commit()
}

// compile-time check whether the StateUpdater implements the Planner interface.
var _ orchestrator.Planner = (*StateUpdater)(nil)
