package planner

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type LoggingPlanner struct {
}

func NewLoggingPlanner() *LoggingPlanner {
	return &LoggingPlanner{}
}

func (s *LoggingPlanner) Process(ctx context.Context, plan *models.Plan) error {
	dict := zerolog.Dict()
	for key, value := range plan.Event.Details {
		dict = dict.Str(key, value)
	}

	level := zerolog.TraceLevel
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

	log.Ctx(ctx).WithLevel(level).Dict("Details", dict).Str("Event", plan.Event.Message).Str("JobID", plan.Job.ID).Msg(message)
	return nil
}

// compile-time check whether the LoggingPlanner implements the Planner interface.
var _ orchestrator.Planner = (*LoggingPlanner)(nil)
