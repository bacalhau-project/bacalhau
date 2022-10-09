package job

import (
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

func VerifyJob(ctx context.Context, j *model.Job) error {
	if reflect.DeepEqual(model.Spec{}, j.Spec) {
		return fmt.Errorf("job spec is empty")
	}

	if reflect.DeepEqual(model.Deal{}, j.Deal) {
		return fmt.Errorf("job deal is empty")
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

	if j.Deal.Confidence > j.Deal.Concurrency {
		return fmt.Errorf("the deal confidence cannot be higher than the concurrency")
	}

	for _, inputVolume := range j.Spec.Inputs {
		if !model.IsValidStorageSourceType(inputVolume.StorageSource) {
			return fmt.Errorf("invalid input volume type: %s", inputVolume.StorageSource.String())
		}
	}

	return nil
}
