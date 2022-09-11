package scenarios

import (
	"context"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/rs/zerolog/log"
)

func SubmitWithConcurrency(ctx context.Context, client *publicapi.APIClient) error {
	jobSpec, jobDeal := getSampleDockerJob()
	jobDeal.Concurrency = 3
	submittedJob, err := client.Submit(ctx, jobSpec, jobDeal, nil)
	log.Info().Msgf("submitted job: %s", submittedJob.ID)

	err = waitUntilCompleted(ctx, client, submittedJob)
	if err != nil {
		return err
	}
	return nil
}
