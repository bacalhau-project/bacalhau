package scenarios

import (
	"context"
	"fmt"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
)

func List(ctx context.Context, client *publicapi.APIClient) error {
	jobs, err := client.List(ctx)
	if err != nil {
		return err
	}

	count := 0
	for _, j := range jobs {
		fmt.Printf("Job: %s\n", j.ID)
		count++
		if count > 10 {
			break
		}
	}
	return nil
}
