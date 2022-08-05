package job

import (
	"fmt"
	"reflect"

	"github.com/filecoin-project/bacalhau/pkg/executor"
)

func VerifyJob(spec executor.JobSpec, deal executor.JobDeal) error {
	if reflect.DeepEqual(executor.JobSpec{}, spec) {
		return fmt.Errorf("job spec is empty")
	}

	if reflect.DeepEqual(executor.JobDeal{}, deal) {
		return fmt.Errorf("job spec is empty")
	}

	return nil
}
