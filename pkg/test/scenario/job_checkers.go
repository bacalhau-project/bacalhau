package scenario

import (
	"github.com/filecoin-project/bacalhau/pkg/job"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

// WaitUntilSuccessful returns a set of job.CheckStatesFunctions that will wait
// until the job they are checking reaches the Completed state on the passed
// number of nodes. The checks will fail if any job errors.
func WaitUntilSuccessful(nodes int) []job.CheckStatesFunction {
	return []job.CheckStatesFunction{
		job.WaitExecutionsThrowErrors([]model.ExecutionStateType{
			model.ExecutionStateFailed,
		}),
		job.WaitForExecutionStates(map[model.ExecutionStateType]int{
			model.ExecutionStateCompleted: nodes,
		}),
	}
}
