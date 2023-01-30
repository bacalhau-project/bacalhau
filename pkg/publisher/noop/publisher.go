package noop

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

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

// Compile-time check that Publisher implements the correct interface:
var _ publisher.Publisher = (*NoopPublisher)(nil)
