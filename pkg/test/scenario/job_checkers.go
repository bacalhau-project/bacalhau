package scenario

import (
	legacy_job "github.com/bacalhau-project/bacalhau/pkg/legacyjob"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

// WaitUntilSuccessful returns a set of job.CheckStatesFunctions that will wait
// until the job they are checking reaches the Completed state on the passed
// number of nodes. The checks will fail if any job errors.
func WaitUntilSuccessful(nodes int) []legacy_job.CheckStatesFunction {
	return []legacy_job.CheckStatesFunction{
		legacy_job.WaitExecutionsThrowErrors([]model.ExecutionStateType{
			model.ExecutionStateFailed,
		}),
		legacy_job.WaitForExecutionStates(map[model.ExecutionStateType]int{
			model.ExecutionStateCompleted: nodes,
		}),
	}
}
