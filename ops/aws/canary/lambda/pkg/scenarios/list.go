package scenarios

import (
	"context"

	"github.com/rs/zerolog/log"

	cmdutil "github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

func List(ctx context.Context, cfg types.BacalhauConfig) error {
	// intentionally delay creation of the client so a new client is created for each
	// scenario to mimic the behavior of bacalhau cli.
	client, err := cmdutil.GetAPIClient(cfg)
	if err != nil {
		return err
	}

	jobs, err := client.List(ctx, "", model.IncludeAny, model.ExcludeNone, 10, false, "created_at", true)
	if err != nil {
		return err
	}
	log.Info().Msgf("listed %d jobs", len(jobs))
	return nil
}
