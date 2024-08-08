package v2

// Telemetry represents the configuration for telemetry components,
// including logging, metrics, and tracing.
type Telemetry struct {
	// Logging is the configuration for logging settings.
	Logging Logging
	// Metrics is the configuration for OpenTelemetry (OTel) metrics settings.
	Metrics Metrics
	// Tracing is the configuration for OpenTelemetry (OTel) tracing settings.
	Tracing Tracing
}

// Logging represents the configuration settings for logging.
type Logging struct {
	// Level specifies the logging level (one of: "trace" "debug", "info", "warn", "error", "fatal", "panic").
	Level string
	// Format specifies the format of the logs (one of:., "console", "color", or "json").
	Format string
}

// Metrics represents the configuration settings for OpenTelemetry (OTel) metrics collection.
type Metrics struct {
	// Endpoint specifies the OpenTelemetry (OTel) endpoint URL for metrics collection.
	Endpoint string
}

// Tracing represents the configuration settings for OpenTelemetry (OTel) tracing.
type Tracing struct {
	// Endpoint specifies the OpenTelemetry (OTel) endpoint URL for tracing data.
	Endpoint string
}
