package verifier

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/storage"
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

	// once we've decided that everything is completed, decide which shards
	// to combine to form a complete result set
	// we will have a list of storage specs that can be downloaded by a client
	// using the appropriate storage driver
	// if the job is deemed to not be finished - this will error
	// individual shards might have errored but if all shards have run,
	// then this will attempt to combine them into a complete result set
	CombineShards(
		ctx context.Context,
		jobState string,
	) ([]storage.StorageSpec, error)
}
