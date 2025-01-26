package planner

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
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
	return &StateUpdater{store: store}
}

// Process updates the state of the executions in the plan according to the scheduler's desired state.
func (s *StateUpdater) Process(ctx context.Context, plan *models.Plan) (err error) {
	metrics := telemetry.NewMetricRecorder(
		attribute.String(AttrPlannerType, "state_updater"),
		attribute.String(AttrOutcomeKey, AttrOutcomeSuccess),
	)
	defer func() {
		if err != nil {
			metrics.Error(err)
			metrics.AddAttributes(attribute.String(AttrOutcomeKey, AttrOutcomeFailure))
		}
		metrics.Done(ctx, processDuration)
	}()

	// If there are no new or updated executions
	// and the job state is not being updated, there is nothing to do.
	if s.isEmpty(plan) {
		return nil
	}

	txContext, err := s.store.BeginTx(ctx)
	metrics.Latency(ctx, processPartDuration, AttrOperationPartBeginTx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer txContext.Rollback() //nolint:errcheck

	if err = s.processExecutions(ctx, txContext, plan, metrics); err != nil {
		return err
	}

	if err = s.processJobState(ctx, txContext, plan, metrics); err != nil {
		return err
	}

	if err = s.processEvaluations(ctx, txContext, plan, metrics); err != nil {
		return err
	}

	if err = s.processEvents(ctx, txContext, plan, metrics); err != nil {
		return err
	}

	return txContext.Commit()
}

func (s *StateUpdater) isEmpty(plan *models.Plan) bool {
	return len(plan.NewExecutions) == 0 &&
		len(plan.UpdatedExecutions) == 0 &&
		len(plan.NewEvaluations) == 0 &&
		plan.DesiredJobState.IsUndefined()
}

func (s *StateUpdater) processExecutions(
	ctx context.Context, txContext jobstore.TxContext, plan *models.Plan, metrics *telemetry.MetricRecorder) error {
	// Create new executions
	if len(plan.NewExecutions) > 0 {
		for _, exec := range plan.NewExecutions {
			if err := s.store.CreateExecution(txContext, *exec); err != nil {
				return err
			}
		}
		metrics.Latency(ctx, processPartDuration, AttrOperationPartCreateExec)
	}

	// Update existing executions
	if len(plan.UpdatedExecutions) > 0 {
		for _, u := range plan.UpdatedExecutions {
			if err := s.store.UpdateExecution(txContext, jobstore.UpdateExecutionRequest{
				ExecutionID: u.Execution.ID,
				NewValues: models.Execution{
					DesiredState: models.State[models.ExecutionDesiredStateType]{
						StateType: u.DesiredState,
						Message:   u.Event.Message,
						Details:   u.Event.Details,
					},
					ComputeState: models.State[models.ExecutionStateType]{
						StateType: u.ComputeState,
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
		metrics.Latency(ctx, processPartDuration, AttrOperationPartUpdateExec)
	}

	return nil
}

func (s *StateUpdater) processJobState(
	ctx context.Context, txContext jobstore.TxContext, plan *models.Plan, metrics *telemetry.MetricRecorder) error {
	if !plan.DesiredJobState.IsUndefined() {
		if err := s.store.UpdateJobState(txContext, jobstore.UpdateJobStateRequest{
			JobID:    plan.Job.ID,
			NewState: plan.DesiredJobState,
			Message:  plan.UpdateMessage,
			Condition: jobstore.UpdateJobCondition{
				ExpectedRevision: plan.Job.Revision,
			},
		}); err != nil {
			return err
		}
		metrics.Latency(ctx, processPartDuration, AttrOperationPartUpdateJob)
	}
	return nil
}

// Create follow-up evaluations, if any
func (s *StateUpdater) processEvaluations(
	ctx context.Context, txContext jobstore.TxContext, plan *models.Plan, metrics *telemetry.MetricRecorder) error {
	if len(plan.NewEvaluations) > 0 {
		for _, eval := range plan.NewEvaluations {
			if err := s.store.CreateEvaluation(txContext, *eval); err != nil {
				return err
			}
		}
		metrics.Latency(ctx, processPartDuration, AttrOperationPartCreateEval)
	}
	return nil
}

func (s *StateUpdater) processEvents(
	ctx context.Context, txContext jobstore.TxContext, plan *models.Plan, metrics *telemetry.MetricRecorder) error {
	if len(plan.JobEvents) == 0 && len(plan.ExecutionEvents) == 0 {
		return nil
	}

	if len(plan.JobEvents) > 0 {
		if err := s.store.AddJobHistory(txContext, plan.Job.ID, plan.JobEvents...); err != nil {
			return err
		}
	}

	if len(plan.ExecutionEvents) > 0 {
		for executionID, events := range plan.ExecutionEvents {
			if err := s.store.AddExecutionHistory(txContext, plan.Job.ID, executionID, events...); err != nil {
				return err
			}
		}
	}
	metrics.Latency(ctx, processPartDuration, AttrOperationPartAddEvents)
	return nil
}

// compile-time check whether the StateUpdater implements the Planner interface.
var _ orchestrator.Planner = (*StateUpdater)(nil)
