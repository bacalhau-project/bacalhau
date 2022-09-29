package publisher

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

// Publisher is the interface for publishing results of a job
// The job spec will choose which publisher(s) it wants to use
// (there can be multiple publishers configured)
type Publisher interface {
	// tells you if the required software is installed on this machine
	IsInstalled(context.Context) (bool, error)

	// compute node
	//
	// once the results have been verified we publish them
	// this will result in a "publish" event that will keep track
	// of the details of the storage spec where the results live
	// the returned storage spec might be nill as jobs
	// can have multiple publishers and some publisher
	// implementations don't concern themselves with storage
	// (e.g. notify slack)
	PublishShardResult(
		ctx context.Context,
		shard model.JobShard,
		hostID string,
		shardResultPath string,
	) (model.StorageSpec, error)

	// return a slice of storage specs that when re-assembled
	// constitutes a complete set of results for a job
	// this is basically picking a valid shard result
	// and then re-assembling the storage spec it has associated
	// with it
	// this is NOT actually downloading results simply giving you
	// enough information to do so if you wanted to
	ComposeResultReferences(
		ctx context.Context,
		jobID string,
	) ([]model.StorageSpec, error)
}
