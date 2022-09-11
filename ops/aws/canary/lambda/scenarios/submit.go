package scenarios

import (
	"context"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
)

func Submit(ctx context.Context, client *publicapi.APIClient) error {
	jobSpec, jobDeal := getSampleDockerJob()
	submittedJob, err := client.Submit(ctx, jobSpec, jobDeal, nil)
	err = waitUntilCompleted(ctx, client, submittedJob)
	if err != nil {
		return err
	}
	return nil
}
