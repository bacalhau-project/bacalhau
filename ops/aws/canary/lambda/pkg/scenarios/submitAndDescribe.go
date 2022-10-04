package scenarios

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/cmd/bacalhau"
	"github.com/rs/zerolog/log"
)

// This scenario mainly calls the same client APIs that describe cmd does.
// It doesn't do much validation except making sure the calls don't fail.
// TODO: we should introduce a describe API and move all the logic there instead of cmd/describe.go
func SubmitAnDescribe(ctx context.Context) error {
	// intentionally delay creation of the client so a new client is created for each
	// scenario to mimic the behavior of bacalhau cli.
	client := bacalhau.GetAPIClient()

	j := getSampleDockerJob()
	submittedJob, err := client.Submit(ctx, j.Spec, j.Deal, nil)
	if err != nil {
		return err
	}

	log.Info().Msgf("submitted job: %s", submittedJob.ID)

	_, ok, err := client.Get(ctx, submittedJob.ID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("job not found matching id: %s", submittedJob.ID)
	}

	_, err = client.GetJobState(ctx, submittedJob.ID)
	if err != nil {
		return err
	}

	_, err = client.GetEvents(ctx, submittedJob.ID)
	if err != nil {
		return err
	}

	_, err = client.GetLocalEvents(ctx, submittedJob.ID)
	if err != nil {
		return err
	}

	return nil
}
