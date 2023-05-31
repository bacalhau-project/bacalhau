package combo

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
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

func (c *piggybackedPublisher) ValidateJob(ctx context.Context, j model.Job) error {
	for _, p := range c.publishers {
		if err := p.ValidateJob(ctx, j); err != nil {
			return err
		}
	}
	return nil
}

func (c *piggybackedPublisher) PublishResult(
	ctx context.Context, executionID string, job model.Job, resultPath string,
) (spec.Storage, error) {
	results, err := callAllPublishers(c.publishers, func(p publisher.Publisher) (spec.Storage, error) {
		return p.PublishResult(ctx, executionID, job, resultPath)
	})
	if err != nil {
		return spec.Storage{}, err
	}

	// TODO metadata is required on (all?) some storage specs
	result := results[0]
	if result.Metadata == nil {
		result.Metadata = map[string]string{}
	}
	for _, other := range results[1:] {
		result.Metadata[other.StorageSource.String()] = other.CID
	}

	return result, nil
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
