package tracing

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
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
	ctx, span := telemetry.NewSpan(ctx, telemetry.GetTracer(), fmt.Sprintf("%s.IsInstalled", t.name))
	defer span.End()

	return t.delegate.IsInstalled(ctx)
}

func (t *tracingPublisher) ValidateJob(ctx context.Context, j models.Job) error {
	return t.delegate.ValidateJob(ctx, j)
}

func (t *tracingPublisher) PublishResult(
	ctx context.Context, execution *models.Execution, resultPath string,
) (spec models.SpecConfig, err error) {
	attributes := execution.Job.MetricAttributes()

	ctx, span := telemetry.NewSpan(ctx, telemetry.GetTracer(), fmt.Sprintf("%s.PublishResult", t.name),
		trace.WithAttributes(attributes...))
	defer span.End()

	stopwatch := telemetry.Timer(ctx, jobPublishDurationMilliseconds, attributes...)
	defer func() {
		dur := stopwatch()
		log.Ctx(ctx).Debug().
			Dur("duration", dur).
			Object("spec", &spec).
			Msg("published result")
	}()

	return t.delegate.PublishResult(ctx, execution, resultPath)
}

var _ publisher.Publisher = &tracingPublisher{}
