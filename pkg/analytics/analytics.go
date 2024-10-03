package analytics

import (
	"context"
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

const ProviderKey = "bacalhau-analytics"
const DefaultOtelCollectorEndpoint = "t.bacalhau.org:4317"

const (
	NodeInstallationIDKey = "installation_id"
	NodeInstanceIDKey     = "instance_id"
	NodeIDHashKey         = "node_id_hash"
	NodeTypeKey           = "node_type"
	NodeVersionKey        = "node_version"
)

type Config struct {
	attributes   []attribute.KeyValue
	otlpEndpoint string
}

type Option func(*Config)

func WithEndpoint(endpoint string) Option {
	return func(c *Config) {
		c.otlpEndpoint = endpoint
	}
}

func WithNodeID(id string) Option {
	return func(c *Config) {
		c.attributes = append(c.attributes, attribute.String(NodeIDHashKey, hashString(id)))
	}
}

func WithNodeType(isRequester, isCompute bool) Option {
	return func(c *Config) {
		var typ string
		if isRequester && isCompute {
			typ = "hybrid"
		} else if isRequester {
			typ = "orchestrator"
		} else if isCompute {
			typ = "compute"
		}
		c.attributes = append(c.attributes, attribute.String(NodeTypeKey, typ))
	}
}

func WithInstallationID(id string) Option {
	return func(c *Config) {
		if id != "" {
			c.attributes = append(c.attributes, attribute.String(NodeInstallationIDKey, id))
		}
	}
}

func WithInstanceID(id string) Option {
	return func(c *Config) {
		if id != "" {
			c.attributes = append(c.attributes, attribute.String(NodeInstanceIDKey, id))
		}
	}
}

func WithVersion(bv *models.BuildVersionInfo) Option {
	return func(c *Config) {
		v, err := semver.NewVersion(bv.GitVersion)
		if err != nil {
			// use the version populated via the `ldflags` flag.
			c.attributes = append(c.attributes, attribute.String(NodeVersionKey, version.GITVERSION))
		} else {
			c.attributes = append(c.attributes, attribute.String(NodeVersionKey, v.String()))
		}
	}
}

func SetupAnalyticsProvider(ctx context.Context, opts ...Option) error {
	config := &Config{
		otlpEndpoint: DefaultOtelCollectorEndpoint, // Default endpoint - grpc
		attributes:   make([]attribute.KeyValue, 0),
	}
	// Apply options
	for _, opt := range opts {
		opt(config)
	}

	exporter, err := otlploggrpc.New(ctx,
		otlploggrpc.WithEndpoint(config.otlpEndpoint),
	)
	if err != nil {
		return fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create a new resource with auto-detected host information
	res, err := resource.New(ctx,
		resource.WithOSType(),
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithAttributes(config.attributes...),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)

	global.SetLoggerProvider(loggerProvider)
	return nil
}

func ShutdownAnalyticsProvider(ctx context.Context) error {
	provider, ok := global.GetLoggerProvider().(*sdklog.LoggerProvider)
	if ok {
		if err := provider.Shutdown(ctx); err != nil {
			log.Trace().Err(err).Msg("failed to shutdown analytics log provider")
		}
	}
	return nil
}

func EmitEvent(ctx context.Context, event *Event) {
	record, err := event.ToLogRecord()
	if err != nil {
		log.Trace().Err(err).Str("type", event.Type).Msg("failed to emit event")
	}
	provider := global.GetLoggerProvider().Logger(ProviderKey)
	provider.Emit(ctx, record)
}
