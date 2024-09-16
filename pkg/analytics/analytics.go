package analytics

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

const ProviderKey = "bacalhau-analytics"

const (
	NodeInstallationIDKey = "installation_id"
	NodeInstanceIDKey     = "instance_id"
	NodeIDKey             = "node_id"
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

func WithNodeNodeID(id string) Option {
	return func(c *Config) {
		c.attributes = append(c.attributes, attribute.String(NodeIDKey, id))
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
		c.attributes = append(c.attributes, attribute.String(NodeInstallationIDKey, id))
	}
}

func WithInstanceID(id string) Option {
	return func(c *Config) {
		c.attributes = append(c.attributes, attribute.String(NodeInstanceIDKey, id))
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
		otlpEndpoint: "t.bacalhau.dev:4318", // Default endpoint - http
		attributes:   make([]attribute.KeyValue, 0),
	}
	// Apply options
	for _, opt := range opts {
		opt(config)
	}

	exporter, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpoint(config.otlpEndpoint),
		otlploghttp.WithTLSClientConfig(&tls.Config{}),
	)
	if err != nil {
		return fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create a new resource with auto-detected host information
	res, err := resource.New(ctx,
		resource.WithOS(),
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

type shutdownLoggerProvider interface {
	otellog.LoggerProvider
	Shutdown(ctx context.Context) error
}

func ShutdownAnalyticsProvider(ctx context.Context) error {
	provider, ok := global.GetLoggerProvider().(*sdklog.LoggerProvider)
	if ok {
		if err := provider.ForceFlush(ctx); err != nil {
			log.Trace().Err(err).Msg("failed to flush analytics log provider")
		}
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
