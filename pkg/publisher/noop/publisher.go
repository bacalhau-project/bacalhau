package noop

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

// NoopPublisherProvider is a publisher provider that always return NoopPublisher regardless of requested publisher type
type NoopPublisherProvider struct {
	noopPublisher *NoopPublisher
}

func NewNoopPublisherProvider(noopPublisher *NoopPublisher) *NoopPublisherProvider {
	return &NoopPublisherProvider{
		noopPublisher: noopPublisher,
	}
}

func (s *NoopPublisherProvider) GetPublisher(context.Context, model.Publisher) (publisher.Publisher, error) {
	return s.noopPublisher, nil
}

func (s *NoopPublisherProvider) HasPublisher(ctx context.Context, publisher model.Publisher) bool {
	_, err := s.GetPublisher(ctx, publisher)
	return err == nil
}

type NoopPublisher struct{}

func NewNoopPublisher() *NoopPublisher {
	return &NoopPublisher{}
}

func (publisher *NoopPublisher) IsInstalled(context.Context) (bool, error) {
	return true, nil
}

func (publisher *NoopPublisher) PublishShardResult(
	ctx context.Context,
	_ model.JobShard,
	_ string,
	_ string,
) (model.StorageSpec, error) {
	//nolint:staticcheck,ineffassign
	ctx, span := system.GetTracer().Start(ctx, "pkg/publisher/noop.PublishShardResult")
	defer span.End()

	return model.StorageSpec{}, nil
}

// Compile-time check that Verifier implements the correct interface:
var _ publisher.PublisherProvider = (*NoopPublisherProvider)(nil)
var _ publisher.Publisher = (*NoopPublisher)(nil)
