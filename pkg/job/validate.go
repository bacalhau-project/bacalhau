package job

import (
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

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

	if !model.IsValidEngine(j.Spec.Engine) {
		return fmt.Errorf("invalid executor type: %s", j.Spec.Engine.String())
	}

	if !model.IsValidVerifier(j.Spec.Verifier) {
		return fmt.Errorf("invalid verifier type: %s", j.Spec.Verifier.String())
	}

	if !model.IsValidPublisher(j.Spec.Publisher) {
		return fmt.Errorf("invalid publisher type: %s", j.Spec.Publisher.String())
	}

	if err := j.Spec.Network.IsValid(); err != nil {
		return err
	}

	if j.Spec.Deal.Confidence > j.Spec.Deal.Concurrency {
		return fmt.Errorf("the deal confidence cannot be higher than the concurrency")
	}

	for _, inputVolume := range j.Spec.Inputs {
		if !model.IsValidStorageSourceType(inputVolume.StorageSource) {
			return fmt.Errorf("invalid input volume type: %s", inputVolume.StorageSource.String())
		}
	}

	return nil
}
