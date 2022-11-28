package noop

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

// Publisher provider that always return NoopPublisher regardless of requested publisher type
type NoopPublisherProvider struct {
	noopPublisher *NoopPublisher
}

func NewNoopPublisherProvider(noopPublisher *NoopPublisher) *NoopPublisherProvider {
	return &NoopPublisherProvider{
		noopPublisher: noopPublisher,
	}
}

func (s *NoopPublisherProvider) GetPublisher(ctx context.Context, publisherType model.Publisher) (publisher.Publisher, error) {
	return s.noopPublisher, nil
}

type NoopPublisher struct{}

func NewNoopPublisher() *NoopPublisher {
	return &NoopPublisher{}
}

func (publisher *NoopPublisher) IsInstalled(ctx context.Context) (bool, error) {
	return true, nil
}

func (publisher *NoopPublisher) PublishShardResult(
	ctx context.Context,
	shard model.JobShard,
	hostID string,
	shardResultPath string,
) (model.StorageSpec, error) {
	//nolint:staticcheck,ineffassign
	ctx, span := system.GetTracer().Start(ctx, "pkg/publisher/noop.PublishShardResult")
	defer span.End()

	return model.StorageSpec{}, nil
}

// Compile-time check that Verifier implements the correct interface:
var _ publisher.PublisherProvider = (*NoopPublisherProvider)(nil)
var _ publisher.Publisher = (*NoopPublisher)(nil)
