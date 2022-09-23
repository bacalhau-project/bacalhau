package scenarios

import (
	"context"

	"github.com/filecoin-project/bacalhau/cmd/bacalhau"
	"github.com/rs/zerolog/log"
)

func Submit(ctx context.Context) error {
	// intentionally delay creation of the client so a new client is created for each
	// scenario to mimic the behavior of bacalhau cli.
	client := bacalhau.GetAPIClient()

	jobSpec, jobDeal := getSampleDockerJob()
	submittedJob, err := client.Submit(ctx, jobSpec, jobDeal, nil)
	if err != nil {
		return err
	}

	log.Info().Msgf("submitted job: %s", submittedJob.ID)

	err = waitUntilCompleted(ctx, client, submittedJob)
	if err != nil {
		return err
	}
	return nil
}
