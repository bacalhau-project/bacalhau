package noop

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

type NoopPublisher struct {
	StateResolver *job.StateResolver
}

func NewNoopPublisher(
	ctx context.Context,
	cm *system.CleanupManager,
	resolver *job.StateResolver,
) (*NoopPublisher, error) {
	return &NoopPublisher{
		StateResolver: resolver,
	}, nil
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

func (publisher *NoopPublisher) ComposeResultReferences(
	ctx context.Context,
	jobID string,
) ([]model.StorageSpec, error) {
	//nolint:staticcheck,ineffassign
	ctx, span := system.GetTracer().Start(ctx, "pkg/publisher/noop.ComposeResultReferences")
	defer span.End()

	return []model.StorageSpec{}, nil
}

// Compile-time check that Verifier implements the correct interface:
var _ publisher.Publisher = (*NoopPublisher)(nil)
