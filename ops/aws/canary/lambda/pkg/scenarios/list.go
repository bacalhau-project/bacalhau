package scenarios

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/rs/zerolog/log"
)

func List(ctx context.Context) error {
	// intentionally delay creation of the client so a new client is created for each
	// scenario to mimic the behavior of bacalhau cli.
	client := getClient()

	jobs, err := client.List(ctx, "", model.IncludeAny, model.ExcludeNone, 10, false, "created_at", true)
	if err != nil {
		return err
	}
	log.Info().Msgf("listed %d jobs", len(jobs))
	return nil
}
