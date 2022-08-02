package verifier

import (
	"context"
)

// Verifier is an interface representing something that can verify the results
// of a job.
type Verifier interface {
	// tells you if the required software is installed on this machine
	IsInstalled(context.Context) (bool, error)

	// the executor has completed the job and produced a local folder of results
	// the verifier will now "process" this local folder into the result
	// that will be broadcast back to the network
	// For example, the IPFS verifier publishes a local folder to IPFS and
	// returns the CID
	ProcessShardResults(
		ctx context.Context,
		jobID string,
		shardIndex int,
		resultsPath string,
	) (string, error)
}
