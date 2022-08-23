package noop

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"go.opentelemetry.io/otel/trace"
)

type NoopPublisher struct {
	StateResolver *job.StateResolver
}

func NewNoopPublisher(
	cm *system.CleanupManager,
	resolver *job.StateResolver,
) (*NoopPublisher, error) {
	return &NoopPublisher{
		StateResolver: resolver,
	}, nil
}

func (publisher *NoopPublisher) IsInstalled(ctx context.Context) (bool, error) {
	ctx, span := newSpan(ctx, "IsInstalled")
	defer span.End()
	return true, nil
}

func (publisher *NoopPublisher) PublishShardResult(
	ctx context.Context,
	hostID string,
	jobID string,
	shardIndex int,
	shardResultPath string,
) (storage.StorageSpec, error) {
	ctx, span := newSpan(ctx, "PublishShardResult")
	defer span.End()
	return storage.StorageSpec{}, nil
}

func (publisher *NoopPublisher) ComposeResultReferences(
	ctx context.Context,
	jobID string,
) ([]storage.StorageSpec, error) {
	ctx, span := newSpan(ctx, "ComposeResultSet")
	defer span.End()
	return []storage.StorageSpec{}, nil
}

func newSpan(ctx context.Context, apiName string) (context.Context, trace.Span) {
	return system.Span(ctx, "publisher/noop", apiName)
}

// Compile-time check that Verifier implements the correct interface:
var _ publisher.Publisher = (*NoopPublisher)(nil)
