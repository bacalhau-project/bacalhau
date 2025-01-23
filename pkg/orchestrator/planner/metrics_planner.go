package planner

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
)

// MetricsPlanner records metrics about plan content as they flow through the planner chain.
// It tracks distributions of executions, evaluations, and events without modifying the plan.
type MetricsPlanner struct{}

// NewMetricsPlanner creates a new instance of MetricsPlanner.
func NewMetricsPlanner() *MetricsPlanner {
	return &MetricsPlanner{}
}

// Process records metrics about plan content including executions, jobs, evaluations and events.
func (s *MetricsPlanner) Process(ctx context.Context, plan *models.Plan) error {
	metrics := telemetry.NewMetricRecorder(
		attribute.String(AttrPlannerType, "metrics_planner"),
	)
	defer func() {
		metrics.Count(ctx, processCount)
		metrics.DoneWithoutTotalDuration(ctx)
	}()

	if len(plan.NewExecutions) > 0 {
		metrics.Histogram(ctx, executionsCreated, float64(len(plan.NewExecutions)))
	}
	if len(plan.UpdatedExecutions) > 0 {
		metrics.Histogram(ctx, executionsUpdated, float64(len(plan.UpdatedExecutions)))
	}
	if !plan.DesiredJobState.IsUndefined() {
		metrics.Count(ctx, jobsUpdated)
	}
	if len(plan.NewEvaluations) > 0 {
		metrics.Histogram(ctx, evaluationsCreated, float64(len(plan.NewEvaluations)))
	}
	if len(plan.JobEvents) > 0 {
		metrics.Histogram(ctx, jobEventsAdded, float64(len(plan.JobEvents)))
	}
	if len(plan.ExecutionEvents) > 0 {
		var totalEvents int
		for _, events := range plan.ExecutionEvents {
			totalEvents += len(events)
		}
		metrics.Histogram(ctx, execEventsAdded, float64(totalEvents))
	}

	return nil
}

var _ orchestrator.Planner = (*MetricsPlanner)(nil)
