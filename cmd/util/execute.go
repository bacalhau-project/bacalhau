package util

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags"
	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

//nolint:funlen,gocyclo // Refactor later
func ExecuteJob(ctx context.Context,
	j *model.Job,
	runtimeSettings *flags.RunTimeSettings,
) (*model.Job, error) {

	err := job.VerifyJob(ctx, j)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Job failed to validate.")
		return nil, err
	}

	return GetAPIClient(ctx).Submit(ctx, j)
}
