package util

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/cmd/v1beta2/util/flags"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/model/v1beta2"
	"github.com/bacalhau-project/bacalhau/pkg/requester/publicapi"
)

//nolint:funlen,gocyclo // Refactor later
func ExecuteJob(ctx context.Context,
	j *v1beta2.Job,
	runtimeSettings *flags.RunTimeSettings,
) (*v1beta2.Job, error) {
	var apiClient *publicapi.RequesterAPIClient

	cm := GetCleanupManager(ctx)

	if runtimeSettings.IsLocal {
		stack, errLocalDevStack := devstack.NewDevStackForRunLocal(ctx, cm, 1, capacity.ConvertGPUString(j.Spec.Resources.GPU))
		if errLocalDevStack != nil {
			return nil, errLocalDevStack
		}

		apiServer := stack.Nodes[0].APIServer
		apiClient = publicapi.NewRequesterAPIClient(apiServer.Address, apiServer.Port)
	} else {
		apiClient = GetAPIClient(ctx)
	}

	err := VerifyJob(ctx, j)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Job failed to validate.")
		return nil, err
	}

	return apiClient.Submit(ctx, j)
}
