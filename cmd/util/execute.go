package util

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	legacy_job "github.com/bacalhau-project/bacalhau/pkg/legacyjob"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

//nolint:funlen,gocyclo // Refactor later
func ExecuteJob(ctx context.Context,
	j *model.Job,
	cfg *config.Config,
	runtimeSettings *cliflags.RunTimeSettingsWithDownload,
) (*model.Job, error) {
	err := legacy_job.VerifyJob(ctx, j)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Job failed to validate.")
		return nil, err
	}

	client, err := GetAPIClient(cfg)
	if err != nil {
		return nil, err
	}
	return client.Submit(ctx, j)
}
