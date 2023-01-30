package scenarios

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/selection"
)

func SubmitWithConcurrencyOwnedNodes(ctx context.Context) error {
	// intentionally delay creation of the client so a new client is created for each
	// scenario to mimic the behavior of bacalhau cli.
	client := getClient()

	j := getSampleDockerJob()
	j.Spec.Deal.Concurrency = 3
	j.Spec.NodeSelectors = []model.LabelSelectorRequirement{
		{
			Key:      "owner",
			Operator: selection.Equals,
			Values:   []string{"bacalhau"},
		},
	}
	submittedJob, err := client.Submit(ctx, j)
	if err != nil {
		return err
	}

	log.Info().Msgf("submitted job: %s", submittedJob.Metadata.ID)

	err = waitUntilCompleted(ctx, client, submittedJob)
	if err != nil {
		return err
	}
	return nil
}
