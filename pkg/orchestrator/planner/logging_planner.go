package planner

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
)

const defaultLogLevel = zerolog.TraceLevel

// LoggingPlanner is a debug-focused component that logs the execution of plans.
// It tracks job state transitions, execution lifecycle, and evaluation scheduling
// at different verbosity levels:
//   - Job completions and failures are logged at INFO and WARN respectively
//   - Job state transitions are logged at TRACE level
//   - Execution and evaluation details are logged at TRACE level when enabled
//
// This planner is meant to be used in conjunction with other planners for debugging purposes.
type LoggingPlanner struct{}

func NewLoggingPlanner() *LoggingPlanner {
	return &LoggingPlanner{}
}

func (s *LoggingPlanner) Process(ctx context.Context, plan *models.Plan) error {
	// Log state changes first as they're most important
	s.logJobStateChanges(ctx, plan)

	// Only log detailed execution and evaluation info at trace level
	if zerolog.GlobalLevel() <= defaultLogLevel {
		s.logNewExecutions(ctx, plan)
		s.logExecutionUpdates(ctx, plan)
		s.logEvaluations(ctx, plan)
	}

	return nil
}

func (s *LoggingPlanner) logJobStateChanges(ctx context.Context, plan *models.Plan) {
	if plan.DesiredJobState.IsUndefined() {
		return
	}

	dict := zerolog.Dict()
	var eventMsg string
	for i, event := range plan.JobEvents {
		if i > 0 {
			eventMsg += ". "
		}
		eventMsg += event.Message
		for k, v := range event.Details {
			dict = dict.Str(k, v)
		}
	}

	level := defaultLogLevel
	message := "Job updated"
	switch plan.DesiredJobState {
	case models.JobStateTypeCompleted:
		level = zerolog.InfoLevel
		message = "Job completed successfully"
	case models.JobStateTypeFailed:
		level = zerolog.WarnLevel
		message = "Job failed"
	default:
	}

	logger := log.Ctx(ctx).WithLevel(level).
		Dict("Details", dict).
		Str("Event", eventMsg).
		Str("JobID", plan.Job.ID).
		Str("OldState", plan.Job.State.StateType.String()).
		Str("NewState", plan.DesiredJobState.String()).
		Uint64("OldRevision", plan.Job.Revision)

	if plan.UpdateMessage != "" {
		logger = logger.Str("Reason", plan.UpdateMessage)
	}

	logger.Msg(message)
}

func (s *LoggingPlanner) logNewExecutions(ctx context.Context, plan *models.Plan) {
	for _, exec := range plan.NewExecutions {
		log.Ctx(ctx).WithLevel(defaultLogLevel).
			Str("JobID", plan.Job.ID).
			Str("ExecutionID", exec.ID).
			Str("NodeID", exec.NodeID).
			Int("PartitionIndex", exec.PartitionIndex).
			Str("DesiredState", exec.DesiredState.StateType.String()).
			Str("ComputeState", exec.ComputeState.StateType.String()).
			Msg("New execution created")
	}
}

func (s *LoggingPlanner) logExecutionUpdates(ctx context.Context, plan *models.Plan) {
	for execID, update := range plan.UpdatedExecutions {
		logger := log.Ctx(ctx).WithLevel(defaultLogLevel).
			Str("JobID", update.Execution.JobID).
			Str("ExecutionID", execID).
			Str("NodeID", update.Execution.NodeID).
			Int("PartitionIndex", update.Execution.PartitionIndex).
			Str("OldState", update.Execution.DesiredState.StateType.String()).
			Str("NewState", update.DesiredState.String()).
			Str("OldComputeState", update.Execution.ComputeState.StateType.String()).
			Str("NewComputeState", update.ComputeState.String())

		// Include event details if present
		if update.Event.Message != "" {
			logger = logger.Str("Reason", update.Event.Message)
		}
		for k, v := range update.Event.Details {
			logger = logger.Str(k, v)
		}

		logger.Msg("Execution updated")
	}
}

func (s *LoggingPlanner) logEvaluations(ctx context.Context, plan *models.Plan) {
	for _, eval := range plan.NewEvaluations {
		logger := log.Ctx(ctx).WithLevel(defaultLogLevel).
			Str("JobID", plan.Job.ID).
			Str("EvalID", eval.ID).
			Str("TriggeredBy", eval.TriggeredBy)

		if !eval.WaitUntil.IsZero() {
			logger = logger.Time("WaitUntil", eval.WaitUntil)
		}
		if eval.Comment != "" {
			logger = logger.Str("Reason", eval.Comment)
		}
		logger.Msg("New evaluation")
	}
}

// compile-time check whether the LoggingPlanner implements the Planner interface.
var _ orchestrator.Planner = (*LoggingPlanner)(nil)
