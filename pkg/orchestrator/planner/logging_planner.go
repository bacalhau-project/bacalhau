package planner

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/rs/zerolog/log"
)

type LoggingPlanner struct {
}

func NewLoggingPlanner() *LoggingPlanner {
	return &LoggingPlanner{}
}

func (s *LoggingPlanner) Process(ctx context.Context, plan *models.Plan) error {
	switch plan.DesiredJobState {
	case model.JobStateCompleted:
		log.Info().Msgf("Job %s completed successfully", plan.Job.ID())
	case model.JobStateError:
		log.Error().Msgf("Job %s failed due to `%s`", plan.Job.ID(), plan.Comment)
	default:
	}

	return nil
}

// compile-time check whether the LoggingPlanner implements the Planner interface.
var _ orchestrator.Planner = (*LoggingPlanner)(nil)
