package util

import (
	"context"

	"github.com/rs/zerolog/log"

	legacy_job "github.com/bacalhau-project/bacalhau/pkg/legacyjob"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client"
)

//nolint:funlen,gocyclo // Refactor later
func ExecuteJob(ctx context.Context, j *model.Job, api client.APIClient) (*model.Job, error) {
	err := legacy_job.VerifyJob(ctx, j)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Job failed to validate.")
		return nil, err
	}

	return api.Submit(ctx, j)
}
