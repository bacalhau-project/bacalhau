package noop

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
)

type NoopPublisher struct{}

func NewNoopPublisher() *NoopPublisher {
	return &NoopPublisher{}
}

func (publisher *NoopPublisher) IsInstalled(context.Context) (bool, error) {
	return true, nil
}

func (publisher *NoopPublisher) PublishShardResult(context.Context, model.JobShard, string, string) (model.StorageSpec, error) {
	return model.StorageSpec{}, nil
}

// Compile-time check that Publisher implements the correct interface:
var _ publisher.Publisher = (*NoopPublisher)(nil)
