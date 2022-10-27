package combo

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
)

type piggybackedPublisher struct {
	publishers []publisher.Publisher
}

// NewPiggybackedPublisher will return a new publisher.Publisher that will call `primary` before then calling `carried`.
// An error will be returned if the `carried` publisher fails, otherwise the returned objects will come from the
// `primary` publisher.
func NewPiggybackedPublisher(primary, carried publisher.Publisher) publisher.Publisher {
	return &piggybackedPublisher{
		publishers: []publisher.Publisher{primary, carried},
	}
}

func (c *piggybackedPublisher) IsInstalled(ctx context.Context) (bool, error) {
	installed, err := callAllPublishers(c.publishers, func(p publisher.Publisher) (bool, error) {
		return p.IsInstalled(ctx)
	})
	if err != nil {
		return false, err
	}
	for _, b := range installed {
		if !b {
			return false, nil
		}
	}

	return true, nil
}

func (c *piggybackedPublisher) PublishShardResult(
	ctx context.Context, shard model.JobShard, hostID string, shardResultPath string,
) (model.StorageSpec, error) {
	results, err := callAllPublishers(c.publishers, func(p publisher.Publisher) (model.StorageSpec, error) {
		return p.PublishShardResult(ctx, shard, hostID, shardResultPath)
	})
	if err != nil {
		return model.StorageSpec{}, err
	}
	return results[0], nil
}

func (c *piggybackedPublisher) ComposeResultReferences(ctx context.Context, jobID string) ([]model.StorageSpec, error) {
	specs, err := callAllPublishers(c.publishers, func(p publisher.Publisher) ([]model.StorageSpec, error) {
		return p.ComposeResultReferences(ctx, jobID)
	})
	if err != nil {
		return nil, err
	}
	return specs[0], nil
}

func callAllPublishers[T any](publishers []publisher.Publisher, f func(publisher.Publisher) (T, error)) ([]T, error) {
	var ts []T
	for _, pub := range publishers {
		t, err := f(pub)
		if err != nil {
			return nil, err
		}
		ts = append(ts, t)
	}
	return ts, nil
}
