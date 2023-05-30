package job

import (
	"context"
	"fmt"
	"reflect"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/docker"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/wasm"
)

// VerifyJobCreatePayload verifies the values in a job creation request are legal.
func VerifyJobCreatePayload(ctx context.Context, jc *model.JobCreatePayload) error {
	if jc.ClientID == "" {
		return fmt.Errorf("ClientID is empty")
	}

	if jc.APIVersion == "" {
		return fmt.Errorf("APIVersion is empty")
	}

	return VerifyJob(ctx, &model.Job{
		APIVersion: jc.APIVersion,
		Spec:       *jc.Spec,
	})
}

func VerifyWasmJobCreatePayload(ctx context.Context, jc *model.WasmJobCreatePayload) error {
	if jc.ClientID == "" {
		return fmt.Errorf("ClientID is empty")
	}

	if jc.WasmJob.APIVersion.String() == "" {
		return fmt.Errorf("APIVersion is empty")
	}
	return jc.WasmJob.Validate()
}

func VerifyDockerJobCreatePayload(ctx context.Context, jc *model.DockerJobCreatePayload) error {
	if jc.ClientID == "" {
		return fmt.Errorf("ClientID is empty")
	}

	if jc.DockerJob.APIVersion.String() == "" {
		return fmt.Errorf("APIVersion is empty")
	}
	return jc.DockerJob.Validate()
}

// VerifyJob verifies that job object passed is valid.
func VerifyJob(ctx context.Context, j *model.Job) error {
	if reflect.DeepEqual(model.Spec{}, j.Spec) {
		return fmt.Errorf("job spec is empty")
	}

	if reflect.DeepEqual(model.Deal{}, j.Spec.Deal) {
		return fmt.Errorf("job deal is empty")
	}

	if j.Spec.Deal.Concurrency <= 0 {
		return fmt.Errorf("concurrency must be >= 1")
	}

	if j.Spec.Deal.Confidence < 0 {
		return fmt.Errorf("confidence must be >= 0")
	}

	if j.Spec.Engine.Schema != docker.EngineType ||
		j.Spec.Engine.Schema != wasm.EngineType {
		log.Warn().Msgf("TODO cannot validate custom engine schema: %s", j.Spec.Engine.Schema)
		return fmt.Errorf("invalid executor type: %s", j.Spec.Engine.Schema.String())
	}

	if !model.IsValidVerifier(j.Spec.Verifier) {
		return fmt.Errorf("invalid verifier type: %s", j.Spec.Verifier.String())
	}

	if !model.IsValidPublisher(j.Spec.PublisherSpec.Type) {
		return fmt.Errorf("invalid publisher type: %s", j.Spec.PublisherSpec.Type.String())
	}

	if err := j.Spec.Network.IsValid(); err != nil {
		return err
	}

	if j.Spec.Deal.Confidence > j.Spec.Deal.Concurrency {
		return fmt.Errorf("the deal confidence cannot be higher than the concurrency")
	}

	// TODO technically we no longer need to validate storage sources since they can be used defined.
	// remove this commented after review
	/*
		for _, inputVolume := range j.Spec.Inputs {
			if !model.IsValidStorageSourceType(inputVolume.StorageSource) {
				return fmt.Errorf("invalid input volume type: %s", inputVolume.StorageSource.String())
			}
		}
	*/

	return nil
}
