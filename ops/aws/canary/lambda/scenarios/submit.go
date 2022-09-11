package scenarios

import (
	"context"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/rs/zerolog/log"
)

func Submit(ctx context.Context, client *publicapi.APIClient) error {
	jobSpec, jobDeal := getSampleDockerJob()
	submittedJob, err := client.Submit(ctx, jobSpec, jobDeal, nil)
	log.Info().Msgf("submitted job: %s", submittedJob.ID)

	err = waitUntilCompleted(ctx, client, submittedJob)
	if err != nil {
		return err
	}
	return nil
}
