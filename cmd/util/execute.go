package util

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/secrets"
)

//nolint:funlen,gocyclo // Refactor later
func ExecuteJob(ctx context.Context,
	j *model.Job,
	runtimeSettings *flags.RunTimeSettings,
) (*model.Job, error) {
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

	err := job.VerifyJob(ctx, j)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Job failed to validate.")
		return nil, err
	}

	if j.Spec.Engine == model.EngineDocker {
		err = encryptDockerEnv(ctx, apiClient, j)
		if err != nil {
			log.Ctx(ctx).Err(err).Msg("failed to encrypt docker env")
			return nil, err
		}
	} else if j.Spec.Engine == model.EngineWasm {
		err = encryptWasmEnv(ctx, apiClient, j)
		if err != nil {
			log.Ctx(ctx).Err(err).Msg("failed to encrypt wasm env")
			return nil, err
		}
	}

	return apiClient.Submit(ctx, j)
}

func encryptDockerEnv(ctx context.Context, apiClient *publicapi.RequesterAPIClient, job *model.Job) error {
	if len(job.Spec.Docker.EnvironmentVariables) == 0 {
		return nil
	}

	publicKey, err := apiClient.GetPublicKey(ctx)
	if err != nil {
		return err
	}

	for i, kv := range job.Spec.Docker.EnvironmentVariables {
		spl := strings.Split(kv, "=")

		encryptedVal, err := secrets.Encrypt([]byte(spl[1]), publicKey)
		if err != nil {
			return err
		}

		encryptedString := hex.EncodeToString(encryptedVal)
		job.Spec.Docker.EnvironmentVariables[i] = fmt.Sprintf("%s=ENC[%s]", spl[0], encryptedString)
	}
	return nil
}

func encryptWasmEnv(ctx context.Context, apiClient *publicapi.RequesterAPIClient, job *model.Job) error {
	// job.Spec.Docker.EnvironmentVariables
	return nil
}
