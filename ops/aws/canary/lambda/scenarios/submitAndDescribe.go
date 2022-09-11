package scenarios

import (
	"context"
	"fmt"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/rs/zerolog/log"
)

// This scenario mainly calls the same client APIs that describe cmd does.
// It doesn't do much validation except making sure the calls don't fail.
// TODO: we should introduce a describe API and move all the logic there instead of cmd/describe.go
func SubmitAnDescribe(ctx context.Context, client *publicapi.APIClient) error {
	jobSpec, jobDeal := getSampleDockerJob()
	submittedJob, err := client.Submit(ctx, jobSpec, jobDeal, nil)
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
