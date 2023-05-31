package noop

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
)

type PublisherHandlerIsInstalled func(ctx context.Context) (bool, error)
type PublisherHandlerPublishResult func(
	ctx context.Context, executionID string, job model.Job, resultPath string) (spec.Storage, error)

func ErrorResultPublisher(err error) PublisherHandlerPublishResult {
	return func(ctx context.Context, executionID string, job model.Job, resultPath string) (spec.Storage, error) {
		return spec.Storage{}, err
	}
}

type PublisherExternalHooks struct {
	IsInstalled   PublisherHandlerIsInstalled
	PublishResult PublisherHandlerPublishResult
}

type PublisherConfig struct {
	ExternalHooks PublisherExternalHooks
}

type NoopPublisher struct {
	externalHooks PublisherExternalHooks
}

func NewNoopPublisher() *NoopPublisher {
	return &NoopPublisher{}
}

func NewNoopPublisherWithConfig(config PublisherConfig) *NoopPublisher {
	p := NewNoopPublisher()
	p.externalHooks = config.ExternalHooks
	return p
}

func (publisher *NoopPublisher) IsInstalled(ctx context.Context) (bool, error) {
	if publisher.externalHooks.IsInstalled != nil {
		return publisher.externalHooks.IsInstalled(ctx)
	}
	return true, nil
}

func (publisher *NoopPublisher) ValidateJob(ctx context.Context, j model.Job) error {
	return nil
}

func (publisher *NoopPublisher) PublishResult(
	ctx context.Context, executionID string, job model.Job, resultPath string) (spec.Storage, error) {
	if publisher.externalHooks.PublishResult != nil {
		return publisher.externalHooks.PublishResult(ctx, executionID, job, resultPath)
	}
	return spec.Storage{}, nil
}

// Compile-time check that Publisher implements the correct interface:
var _ publisher.Publisher = (*NoopPublisher)(nil)
