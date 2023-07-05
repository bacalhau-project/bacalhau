package util

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hashicorp/go-multierror"

	"github.com/bacalhau-project/bacalhau/pkg/model/v1beta2"
)

// VerifyJob verifies that job object passed is valid.
func VerifyJob(ctx context.Context, j *v1beta2.Job) error {
	// NB(forrest): this is a great place to use multierror pattern since it will expose everything wrong if there is
	// more than one issue with the job.
	var veriferrs *multierror.Error
	if reflect.DeepEqual(v1beta2.Spec{}, j.Spec) {
		veriferrs = multierror.Append(veriferrs, fmt.Errorf("job spec is empty"))
	}

	if err := j.Spec.Deal.IsValid(); err != nil {
		return err
	}

	if !v1beta2.IsValidEngine(j.Spec.Engine) {
		veriferrs = multierror.Append(veriferrs, fmt.Errorf("invalid executor type: %s", j.Spec.Engine.String()))
	}

	if !v1beta2.IsValidVerifier(j.Spec.Verifier) {
		veriferrs = multierror.Append(veriferrs, fmt.Errorf("invalid verifier type: %s", j.Spec.Verifier.String()))
	}

	if !v1beta2.IsValidPublisher(j.Spec.PublisherSpec.Type) {
		veriferrs = multierror.Append(veriferrs, fmt.Errorf("invalid publisher type: %s", j.Spec.PublisherSpec.Type.String()))
	}

	if err := j.Spec.Network.IsValid(); err != nil {
		veriferrs = multierror.Append(veriferrs, err)
	}

	for _, inputVolume := range j.Spec.Inputs {
		if !v1beta2.IsValidStorageSourceType(inputVolume.StorageSource) {
			veriferrs = multierror.Append(veriferrs, fmt.Errorf("invalid input volume type: %s", inputVolume.StorageSource.String()))
		}
	}

	// TODO(forrest): shouldn't we verify the outputs? Currently if we do now the
	// tests fail as outputs don't have a storage type when specified via a file test
	// such as TestCancelTerminalJob will fail if this is uncommented because the job
	// doesn't have a valid output type (if sourceUnknown)
	/*
		for _, outputVolume := range j.Spec.Outputs {
			if !v1beta2.IsValidStorageSourceType(outputVolume.StorageSource) {
				veriferrs = multierror.Append(veriferrs, fmt.Errorf("invalid output volume type: %s", outputVolume.StorageSource.String()))
			}
		}
	*/

	return veriferrs.ErrorOrNil()
}
