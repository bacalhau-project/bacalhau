package jobstore

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"

	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
)

var (
	Meter = otel.GetMeterProvider().Meter("jobstore")

	// OperationDuration Duration of database operations
	OperationDuration = telemetry.Must(Meter.Float64Histogram(
		semconv.DBClientOperationDurationName,                                // "db.client.operation.duration"
		metric.WithDescription(semconv.DBClientOperationDurationDescription), // "Duration of database operations"
		metric.WithUnit(semconv.DBClientOperationDurationUnit),               // "s"
		metric.WithExplicitBucketBoundaries(telemetry.DurationMsBuckets...),
	))

	// OperationPartDuration Duration of sub-operations within a database operation
	OperationPartDuration = telemetry.Must(Meter.Float64Histogram(
		"db.client.operation.part.duration",
		metric.WithDescription("Duration of sub-operations within a database operation"),
		metric.WithUnit("s"), // Consistent with OperationDuration
		metric.WithExplicitBucketBoundaries(telemetry.DurationMsBuckets...),
	))

	// OperationCount Count of database operations
	OperationCount = telemetry.Must(Meter.Int64Counter(
		"db.client.operation.count",
		metric.WithDescription("Number of database operations performed"),
		metric.WithUnit("1"),
	))

	// RowsRead Rows read from the database
	RowsRead = telemetry.Must(Meter.Int64Counter(
		"db.client.read.rows",
		metric.WithDescription("Total rows read from the database"),
		metric.WithUnit("1"),
	))

	// DataRead Bytes read from the database
	DataRead = telemetry.Must(Meter.Int64Counter(
		"db.client.read.bytes",
		metric.WithDescription("Total bytes read from the database"),
		metric.WithUnit("By"),
	))

	// DataWritten Bytes written to the database
	DataWritten = telemetry.Must(Meter.Int64Counter(
		"db.client.write.bytes",
		metric.WithDescription("Total bytes written to the database"),
		metric.WithUnit("By"),
	))

	// StoreSize Current size of the store
	StoreSize = telemetry.Must(Meter.Int64UpDownCounter(
		"db.store.size",
		metric.WithDescription("Current size of the store"),
		metric.WithUnit("By"),
	))
)

var (
	AttrScopeKey        = attribute.Key("operation_scope")
	AttrNamespaceKey    = attribute.Key("job_namespace")
	AttrFromStateKey    = attribute.Key("from_state")
	AttrToStateKey      = attribute.Key("to_state")
	FromDesiredStateKey = attribute.Key("from_desired_state")
	ToDesiredStateKey   = attribute.Key("to_desired_state")
)

// Common attribute keys for jobstore
const (
	AttrOperationCreate = "create"
	AttrOperationGet    = "get"
	AttrOperationList   = "list"
	AttrOperationUpdate = "update"
	AttrOperationDelete = "delete"

	// Data operations
	AttrOperationPartRead   = "read"
	AttrOperationPartWrite  = "write"
	AttrOperationPartDelete = "delete"

	// Index operations
	AttrOperationPartIndexRead   = "index_read"
	AttrOperationPartIndexWrite  = "index_write"
	AttrOperationPartIndexDelete = "index_delete"

	// Bucket operations
	AttrOperationPartBucketRead   = "bucket_read"
	AttrOperationPartBucketWrite  = "bucket_write"
	AttrOperationPartBucketDelete = "bucket_delete"

	// Processing operations
	AttrOperationPartValidate  = "validate"
	AttrOperationPartMarshal   = "marshal"
	AttrOperationPartUnmarshal = "unmarshal"
	AttrOperationPartReifyID   = "reify_id"

	// Sequence operations
	AttrOperationPartSequence = "sequence"

	// Event operations
	AttrOperationPartEventWrite = "event_write"

	// Scopes
	AttrScopeAll        = "all"
	AttrScopeNamespace  = "namespace"
	AttrScopeInProgress = "in_progress"
	AttrScopeJob        = "job"
	AttrScopeExecution  = "execution"
)
