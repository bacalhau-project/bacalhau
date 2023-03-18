package noop

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
)

type NoopPublisher struct{}

func NewNoopPublisher() *NoopPublisher {
	return &NoopPublisher{}
}

func (publisher *NoopPublisher) IsInstalled(context.Context) (bool, error) {
	return true, nil
}

func (publisher *NoopPublisher) PublishResult(context.Context, model.Job, string, string) (model.StorageSpec, error) {
	return model.StorageSpec{}, nil
}

// Compile-time check that Publisher implements the correct interface:
var _ publisher.Publisher = (*NoopPublisher)(nil)
