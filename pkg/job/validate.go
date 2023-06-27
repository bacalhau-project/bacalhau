package job

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hashicorp/go-multierror"

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
	// NB(forrest): this is a great place to use multierror pattern since it will expose everything wrong if there is
	// more than one issue with the job.
	var veriferrs *multierror.Error
	if reflect.DeepEqual(model.Spec{}, j.Spec) {
		veriferrs = multierror.Append(veriferrs, fmt.Errorf("job spec is empty"))
	}

	if err := j.Spec.Deal.IsValid(); err != nil {
		return err
	}

	if !model.IsValidEngine(j.Spec.Engine) {
		veriferrs = multierror.Append(veriferrs, fmt.Errorf("invalid executor type: %s", j.Spec.Engine.String()))
	}

	if !model.IsValidVerifier(j.Spec.Verifier) {
		veriferrs = multierror.Append(veriferrs, fmt.Errorf("invalid verifier type: %s", j.Spec.Verifier.String()))
	}

	if !model.IsValidPublisher(j.Spec.PublisherSpec.Type) {
		veriferrs = multierror.Append(veriferrs, fmt.Errorf("invalid publisher type: %s", j.Spec.PublisherSpec.Type.String()))
	}

	if err := j.Spec.Network.IsValid(); err != nil {
		veriferrs = multierror.Append(veriferrs, err)
	}

	for _, inputVolume := range j.Spec.Inputs {
		if !model.IsValidStorageSourceType(inputVolume.StorageSource) {
			veriferrs = multierror.Append(veriferrs, fmt.Errorf("invalid input volume type: %s", inputVolume.StorageSource.String()))
		}
	}

	// TODO(forrest): shouldn't we verify the outputs? Currently if we do now the
	// tests fail as outputs don't have a storage type when specified via a file test
	// such as TestCancelTerminalJob will fail if this is uncommented because the job
	// doesn't have a valid output type (if sourceUnknown)
	/*
		for _, outputVolume := range j.Spec.Outputs {
			if !model.IsValidStorageSourceType(outputVolume.StorageSource) {
				veriferrs = multierror.Append(veriferrs, fmt.Errorf("invalid output volume type: %s", outputVolume.StorageSource.String()))
			}
		}
	*/

	return veriferrs.ErrorOrNil()
}
