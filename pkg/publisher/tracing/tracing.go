package tracing

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/util/reflection"
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

func (t *tracingPublisher) ValidateJob(ctx context.Context, j model.Job) error {
	return t.delegate.ValidateJob(ctx, j)
}

func (t *tracingPublisher) PublishResult(
	ctx context.Context, executionID string, j model.Job, resultPath string,
) (spec.Storage, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), fmt.Sprintf("%s.PublishResult", t.name))
	defer span.End()

	return t.delegate.PublishResult(ctx, executionID, j, resultPath)
}

var _ publisher.Publisher = &tracingPublisher{}
