package verifier

import (
	"github.com/filecoin-project/bacalhau/pkg/types"
)

type Verifier interface {

	// tells you if the required software is installed on this machine
	IsInstalled() (bool, error)

	// the executor has completed the job and produced a local folder of results
	// the verifier will now "process" this local folder into the result
	// that will be broadcast back to the network
	ProcessResultsFolder(
		job *types.Job,
		resultsFolder string,
	) (string, error)
}
