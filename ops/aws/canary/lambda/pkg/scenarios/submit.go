package scenarios

import (
	"context"

	"github.com/rs/zerolog/log"
)

func Submit(ctx context.Context) error {
	// intentionally delay creation of the client so a new client is created for each
	// scenario to mimic the behavior of bacalhau cli.
	client := getClient()

	j, err := getSampleDockerJob()
	if err != nil {
		return err
	}
	submittedJob, err := client.Submit(ctx, j)
	if err != nil {
		return err
	}

	log.Info().Msgf("submitted job: %s", submittedJob.Metadata.ID)

	err = waitUntilCompleted(ctx, client, submittedJob)
	if err != nil {
		return err
	}
	return nil
}
