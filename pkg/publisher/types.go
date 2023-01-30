package publisher

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

// PublisherProvider returns a publisher for the given publisher type
type PublisherProvider interface {
	model.Provider[model.Publisher, Publisher]
}

// Publisher is the interface for publishing results of a job
// The job spec will choose which publisher(s) it wants to use
// (there can be multiple publishers configured)
type Publisher interface {
	model.Providable

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
}
