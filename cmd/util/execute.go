package util

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/requester/publicapi"
)

//nolint:funlen,gocyclo // Refactor later
func ExecuteJob(ctx context.Context,
	j *model.Job,
	runtimeSettings *flags.RunTimeSettings,
) (*model.Job, error) {
	var apiClient *publicapi.RequesterAPIClient

	cm := GetCleanupManager(ctx)

	if runtimeSettings.IsLocal {
		stack, err := devstack.Setup(ctx, cm,
			devstack.WithNumberOfHybridNodes(1),
			devstack.WithPublicIPFSMode(true),
			devstack.WithComputeConfig(node.ComputeConfig{
				TotalResourceLimits: model.ResourceUsageData{
					GPU: capacity.ConvertGPUString(j.Spec.Resources.GPU),
				},
				JobSelectionPolicy: model.JobSelectionPolicy{
					Locality:            model.Anywhere,
					RejectStatelessJobs: true,
				},
			}),
		)
		if err != nil {
			return nil, err
		}

		apiServer := stack.Nodes[0].APIServer
		apiClient = publicapi.NewRequesterAPIClient(apiServer.Address, apiServer.Port)
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
