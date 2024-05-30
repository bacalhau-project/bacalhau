package scenarios

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	cmdutil "github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels/legacymodels"
)

// This scenario mainly calls the same client APIs that describe cmd does.
// It doesn't do much validation except making sure the calls don't fail.
// TODO: we should introduce a describe API and move all the logic there instead of cmd/describe.go
func SubmitAnDescribe(ctx context.Context, cfg types.BacalhauConfig) error {
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
	submittedJob, err := apiV1.Submit(ctx, j)
	if err != nil {
		return err
	}

	log.Info().Msgf("submitted job: %s", submittedJob.Metadata.ID)

	_, ok, err := apiV1.Get(ctx, submittedJob.Metadata.ID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("job not found matching id: %s", submittedJob.Metadata.ID)
	}

	_, err = apiV1.GetJobState(ctx, submittedJob.Metadata.ID)
	if err != nil {
		return err
	}

	_, err = apiV1.GetEvents(ctx, submittedJob.Metadata.ID, legacymodels.EventFilterOptions{})
	if err != nil {
		return err
	}

	return nil
}
