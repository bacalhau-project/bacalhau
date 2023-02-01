package telemetry

// Environment Variables
const (
	// environment variable that defines the endpoint for the oltp collector
	// e.g. http://localhost:4318 for an insecure local collector
	otlpEndpoint = "OTEL_EXPORTER_OTLP_ENDPOINT"

	// defines the protocol used to push to oltp collector. both http/protobuf (default) and grpc are supported.
	otlpProtocol = "OTEL_EXPORTER_OTLP_PROTOCOL"

	// allows defining a different oltp collector for traces
	otlpTracesEndpoint = "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"

	// allows defining a different oltp protocol for traces
	otlpTracesProtocol = "OTEL_EXPORTER_OTLP_TRACES_PROTOCOL"

	// allows defining a different oltp collector for metrics
	otlpMetricsEndpoint = "OTEL_EXPORTER_OTLP_METRICS_ENDPOINT"

	// allows defining a different oltp protocol for metrics
	otlpMetricsProtocol = "OTEL_EXPORTER_OTLP_METRICS_PROTOCOL"
)

// Constants
const (
	otlpProtocolHTTP = "http/protobuf"

	otlpProtocolGrpc = "grpc"
)
