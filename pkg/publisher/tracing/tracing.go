package tracing

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/reflection"
)

type tracingPublisher struct {
	delegate publisher.Publisher
	name     string
}

func Wrap(delegate publisher.Publisher) publisher.Publisher {
	return &tracingPublisher{
		delegate: delegate,
		name:     reflection.StructName(delegate),
	}
}

func (t *tracingPublisher) IsInstalled(ctx context.Context) (bool, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), fmt.Sprintf("%s.IsInstalled", t.name))
	defer span.End()

	return t.delegate.IsInstalled(ctx)
}

func (t *tracingPublisher) PublishShardResult(
	ctx context.Context, shard model.JobShard, hostID string, shardResultPath string,
) (model.StorageSpec, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), fmt.Sprintf("%s.PublishShardResult", t.name))
	defer span.End()

	return t.delegate.PublishShardResult(ctx, shard, hostID, shardResultPath)
}

var _ publisher.Publisher = &tracingPublisher{}
