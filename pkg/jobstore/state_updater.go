package jobstore

import (
	context "context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/util"
	"github.com/rs/zerolog/log"
)

func MakeEvaluationStateUpdater(ctx context.Context, store Store) func(*models.Evaluation) {
	detachedContext := util.NewDetachedContext(ctx)
	return func(e *models.Evaluation) {
		log.Ctx(ctx).Trace().Str("EvaluationID", e.ID).Str("NewStatus", e.Status).Msg("Updating stored state")

		request := UpdateEvaluationRequest{
			JobID:            e.JobID,
			EvaluationID:     e.ID,
			NewStatus:        e.Status,
			ModificationTime: e.ModifyTime,
			Revision:         e.Revision,
		}

		err := store.UpdateEvaluation(detachedContext, request)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("Failed to update evaluation state in jobstore")
		}
	}
}
