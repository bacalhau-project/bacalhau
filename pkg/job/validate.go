package job

import (
	"fmt"
	"reflect"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

func VerifyJob(j *model.Job) error {
	if reflect.DeepEqual(model.Spec{}, j.Spec) {
		return fmt.Errorf("job spec is empty")
	}

	if reflect.DeepEqual(model.Deal{}, j.Deal) {
		return fmt.Errorf("job deal is empty")
	}

	if !model.IsValidEngineType(j.Spec.Engine) {
		return fmt.Errorf("invalid executor type: %s", j.Spec.Engine.String())
	}

	if !model.IsValidVerifierType(j.Spec.Verifier) {
		return fmt.Errorf("invalid verifier type: %s", j.Spec.Verifier.String())
	}

	if !model.IsValidPublisherType(j.Spec.Publisher) {
		return fmt.Errorf("invalid publisher type: %s", j.Spec.Publisher.String())
	}

	if j.Deal.Confidence > j.Deal.Concurrency {
		return fmt.Errorf("the deal confidence cannot be higher than the concurrency")
	}

	for _, inputVolume := range j.Spec.Inputs {
		if !model.IsValidStorageSourceType(inputVolume.Engine) {
			return fmt.Errorf("invalid input volume type: %s", inputVolume.Engine.String())
		}
	}

	return nil
}
