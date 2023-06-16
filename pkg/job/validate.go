package job

import (
	"context"
	"fmt"
	"reflect"

	"github.com/bacalhau-project/bacalhau/pkg/model"
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

// VerifyJob verifies that job object passed is valid.
func VerifyJob(ctx context.Context, j *model.Job) error {
	if reflect.DeepEqual(model.Spec{}, j.Spec) {
		return fmt.Errorf("job spec is empty")
	}

	if err := j.Spec.Deal.IsValid(); err != nil {
		return err
	}

	if !model.IsValidEngine(j.Spec.Engine) {
		return fmt.Errorf("invalid executor type: %s", j.Spec.Engine.String())
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

	for _, inputVolume := range j.Spec.Inputs {
		if !model.IsValidStorageSourceType(inputVolume.StorageSource) {
			return fmt.Errorf("invalid input volume type: %s", inputVolume.StorageSource.String())
		}
	}

	return nil
}
