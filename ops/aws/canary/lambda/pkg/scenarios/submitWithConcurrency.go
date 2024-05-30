package scenarios

import (
	"context"

	"github.com/rs/zerolog/log"

	cmdutil "github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

func SubmitWithConcurrency(ctx context.Context, cfg types.BacalhauConfig) error {
	// intentionally delay creation of the client so a new client is created for each
	// scenario to mimic the behavior of bacalhau cli.
	apiV1, err := cmdutil.GetAPIClient(cfg)
	if err != nil {
		return err
	}

	j, err := getSampleDockerJob()
	if err != nil {
		return err
	}
	j.Spec.Deal.Concurrency = 3
	submittedJob, err := apiV1.Submit(ctx, j)
	if err != nil {
		return err
	}

	log.Info().Msgf("submitted job: %s", submittedJob.Metadata.ID)

	err = waitUntilCompleted(ctx, apiV1, submittedJob)
	if err != nil {
		return err
	}
	return nil
}
