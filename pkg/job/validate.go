package job

import (
	"fmt"
	"reflect"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
)

func VerifyJob(spec executor.JobSpec, deal executor.JobDeal) error {
	if reflect.DeepEqual(executor.JobSpec{}, spec) {
		return fmt.Errorf("job spec is empty")
	}

	if reflect.DeepEqual(executor.JobDeal{}, deal) {
		return fmt.Errorf("job spec is empty")
	}

	if !executor.IsValidEngineType(spec.Engine) {
		return fmt.Errorf("invalid executor type: %s", spec.Engine.String())
	}

	if !verifier.IsValidVerifierType(spec.Verifier) {
		return fmt.Errorf("invalid verifier type: %s", spec.Verifier.String())
	}

	if !publisher.IsValidPublisherType(spec.Publisher) {
		return fmt.Errorf("invalid publisher type: %s", spec.Publisher.String())
	}

	if deal.Confidence > deal.Concurrency {
		return fmt.Errorf("the deal confidence cannot be higher than the concurrency")
	}

	for _, inputVolume := range spec.Inputs {
		if !storage.IsValidStorageSourceType(inputVolume.Engine) {
			return fmt.Errorf("invalid input volume type: %s", inputVolume.Engine.String())
		}
	}

	return nil
}
