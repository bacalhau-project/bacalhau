package scenarios

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/cmd/bacalhau"
)

func List(ctx context.Context) error {
	// intentionally delay creation of the client so a new client is created for each
	// scenario to mimic the behavior of bacalhau cli.
	client := bacalhau.GetAPIClient()

	jobs, err := client.List(ctx)
	if err != nil {
		return err
	}

	count := 0
	for i := range jobs {
		fmt.Printf("Job: %s\n", jobs[i].ID)
		count++
		if count > 10 { //nolint:gomnd
			break
		}
	}
	return nil
}
