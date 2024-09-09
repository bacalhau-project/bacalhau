package analytics

import (
	"context"

	otellog "go.opentelemetry.io/otel/log"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

var _ Recorder = (*NoopRecorder)(nil)

type NoopRecorder struct{}

func (n *NoopRecorder) Stop(ctx context.Context) error {
	return nil
}

func (n *NoopRecorder) EmitEvent(ctx context.Context, event EventType, properties ...otellog.KeyValue) {
	return
}

func (n *NoopRecorder) EmitJobEvent(ctx context.Context, event EventType, j models.Job) {
	return
}
