package util

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/node"
)

//nolint:funlen,gocyclo // Refactor later
func ExecuteJob(ctx context.Context,
	j *model.Job,
	runtimeSettings *flags.RunTimeSettings,
) (*model.Job, error) {
	var apiClient *client.APIClient

	cm := GetCleanupManager(ctx)

	if runtimeSettings.IsLocal {
		stack, err := devstack.Setup(ctx, cm,
			devstack.WithNumberOfHybridNodes(1),
			devstack.WithPublicIPFSMode(true),
			devstack.WithComputeConfig(node.ComputeConfig{
				TotalResourceLimits: models.Resources{
					GPU: model.ConvertGPUString(j.Spec.Resources.GPU),
				},
				JobSelectionPolicy: node.JobSelectionPolicy{
					Locality:            semantic.Anywhere,
					RejectStatelessJobs: true,
				},
			}),
		)
		if err != nil {
			return nil, err
		}

		apiServer := stack.Nodes[0].APIServer
		apiClient = client.NewAPIClient(apiServer.Address, apiServer.Port)
	} else {
		apiClient = GetAPIClient(ctx)
	}

	err := job.VerifyJob(ctx, j)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Job failed to validate.")
		return nil, err
	}

	return apiClient.Submit(ctx, j)
}
