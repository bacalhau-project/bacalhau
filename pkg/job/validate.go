package job

import (
	"fmt"
	"reflect"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

func VerifyJob(spec model.JobSpec, deal model.JobDeal) error {
	if reflect.DeepEqual(model.JobSpec{}, spec) {
		return fmt.Errorf("job spec is empty")
	}

	if reflect.DeepEqual(model.JobDeal{}, deal) {
		return fmt.Errorf("job spec is empty")
	}

	if !model.IsValidEngineType(spec.Engine) {
		return fmt.Errorf("invalid executor type: %s", spec.Engine.String())
	}

	if !model.IsValidVerifierType(spec.Verifier) {
		return fmt.Errorf("invalid verifier type: %s", spec.Verifier.String())
	}

	if !model.IsValidPublisherType(spec.Publisher) {
		return fmt.Errorf("invalid publisher type: %s", spec.Publisher.String())
	}

	if deal.Confidence > deal.Concurrency {
		return fmt.Errorf("the deal confidence cannot be higher than the concurrency")
	}

	for _, inputVolume := range spec.InputVolumes {
		if !model.IsValidStorageSourceType(inputVolume.Engine) {
			return fmt.Errorf("invalid input volume type: %s", inputVolume.Engine.String())
		}
	}

	return nil
}
