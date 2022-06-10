package verifier

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/types"
)

// Verifier is an interface representing something that can verify the results
// of a job.
type Verifier interface {
	// tells you if the required software is installed on this machine
	IsInstalled(ctx context.Context) (bool, error)

	// the executor has completed the job and produced a local folder of results
	// the verifier will now "process" this local folder into the result
	// that will be broadcast back to the network
	ProcessResultsFolder(ctx context.Context, job *types.Job,
		resultsFolder string) (string, error)
}
