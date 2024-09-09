package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type Recorder interface {
	EmitEvent(ctx context.Context, event EventType, properties ...otellog.KeyValue)
	EmitJobEvent(ctx context.Context, event EventType, j models.Job)
	Stop(ctx context.Context) error
}

var _ Recorder = (*LogRecorder)(nil)

type LogRecorder struct {
	provider *sdklog.LoggerProvider
}

func New(ctx context.Context, opts ...Option) (*LogRecorder, error) {
	config := &Config{
		otlpEndpoint: "localhost:4317", // Default endpoint
		attributes:   make([]attribute.KeyValue, 0),
	}
	// Apply options
	for _, opt := range opts {
		if err := opt(config); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	// Create the file exporter
	// TODO before merging we'll need to disable this
	stdoutExporter, err := stdoutlog.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout exporter: %w", err)
	}

	exporter, err := otlploggrpc.New(ctx,
		otlploggrpc.WithEndpoint(config.otlpEndpoint), otlploggrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create a new resource with auto-detected host information
	res, err := resource.New(ctx,
		resource.WithOS(),
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithAttributes(config.attributes...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(stdoutExporter)),
	)

	return &LogRecorder{
		provider: loggerProvider,
	}, nil
}

func (a *LogRecorder) Stop(ctx context.Context) error {
	defer func() {
		if err := a.provider.Shutdown(ctx); err != nil {
			log.Warn().Err(err).Msg("failed to shutdown analytics")
		}
	}()
	if err := a.provider.ForceFlush(ctx); err != nil {
		return fmt.Errorf("failed to flush analytics: %w", err)
	}
	return nil
}

type EventType string

const (
	JobComplete EventType = "job_complete"
)

const (
	EventKey      = "event"
	PropertiesKey = "properties"
)

func (a *LogRecorder) EmitEvent(ctx context.Context, event EventType, properties ...otellog.KeyValue) {
	record := otellog.Record{}
	record.SetTimestamp(time.Now().UTC())
	record.AddAttributes(
		otellog.String(EventKey, string(event)),
		otellog.Map(PropertiesKey, properties...),
	)
	a.provider.Logger("bacalhau-analytics").Emit(ctx, record)
}

func (a *LogRecorder) EmitJobEvent(ctx context.Context, event EventType, j models.Job) {
	jobAttributes := makeJobAttributes(j)
	taskAttributes := makeTaskAttributes(j.Task())
	a.EmitEvent(ctx, event, append(jobAttributes, taskAttributes...)...)
}
