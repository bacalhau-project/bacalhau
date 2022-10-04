package scenarios

import (
	"context"
	"github.com/rs/zerolog/log"

	"github.com/filecoin-project/bacalhau/cmd/bacalhau"
)

func List(ctx context.Context) error {
	// intentionally delay creation of the client so a new client is created for each
	// scenario to mimic the behavior of bacalhau cli.
	client := bacalhau.GetAPIClient()

	jobs, err := client.List(ctx, "", 10, false, "created_at", true)
	if err != nil {
		return err
	}
	log.Info().Msgf("listed %d jobs", len(jobs))
	return nil
}
