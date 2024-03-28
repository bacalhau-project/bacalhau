package planner

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
)

// ReEvaluator is a planner implementation that emits events based on the job state.
type ReEvaluator struct {
	id               string
	evaluationBroker orchestrator.EvaluationBroker
}

// ReEvaluatorParams holds the parameters for creating a new ReEvaluator.
type ReEvaluatorParams struct {
	ID               string
	EvaluationBroker orchestrator.EvaluationBroker
}

// NewReEvaluator creates a new instance of ReEvaluator.
func NewReEvaluator(params ReEvaluatorParams) *ReEvaluator {
	return &ReEvaluator{
		id:               params.ID,
		evaluationBroker: params.EvaluationBroker,
	}
}

// Process creates new evaluations, if the plan demands it
func (s *ReEvaluator) Process(ctx context.Context, plan *models.Plan) error {
	if plan.NewEvaluation != nil {
		return s.evaluationBroker.Enqueue(plan.NewEvaluation)
	}
	return nil
}

// compile-time check whether the ReEvaluator implements the Planner interface.
var _ orchestrator.Planner = (*ReEvaluator)(nil)
