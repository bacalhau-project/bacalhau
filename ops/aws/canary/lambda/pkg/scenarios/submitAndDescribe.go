package scenarios

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
)

// This scenario mainly calls the same client APIs that describe cmd does.
// It doesn't do much validation except making sure the calls don't fail.
// TODO: we should introduce a describe API and move all the logic there instead of cmd/describe.go
func SubmitAnDescribe(ctx context.Context) error {
	// intentionally delay creation of the client so a new client is created for each
	// scenario to mimic the behavior of bacalhau cli.
	client := getClient()

	j := getSampleDockerJob()
	submittedJob, err := client.Submit(ctx, j)
	if err != nil {
		return err
	}

	log.Info().Msgf("submitted job: %s", submittedJob.Metadata.ID)

	_, ok, err := client.Get(ctx, submittedJob.Metadata.ID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("job not found matching id: %s", submittedJob.Metadata.ID)
	}

	_, err = client.GetJobState(ctx, submittedJob.Metadata.ID)
	if err != nil {
		return err
	}

	_, err = client.GetEvents(ctx, submittedJob.Metadata.ID)
	if err != nil {
		return err
	}

	return nil
}
