package telemetry

// BucketBoundaries defines standard histogram bucket boundaries for different types of measurements
//
// Example usage:
//
// var OperationDuration = telemetry.Must(Meter.Float64Histogram(
//     "operation.duration",
//     metric.WithDescription("Duration of operation"),
//     metric.WithUnit("s"),
//     metric.WithExplicitBucketBoundaries(DurationMsBuckets...),
// ))
//
// var PayloadSize = telemetry.Must(Meter.Float64Histogram(
//     "payload.size",
//     metric.WithDescription("Size of payload"),
//     metric.WithUnit("By"),
//     metric.WithExplicitBucketBoundaries(BytesBuckets...),
// ))

var (
	// DurationMsBuckets defines boundaries for millisecond-scale operations
	// Covers spans from 1ms to 10s with exponential growth
	// Useful for API calls, quick DB operations, network requests
	DurationMsBuckets = []float64{
		0.001, // 1ms
		0.005, // 5ms
		0.01,  // 10ms
		0.05,  // 50ms
		0.1,   // 100ms
		0.5,   // 500ms
		1,     // 1s
		5,     // 5s
		10,    // 10s
	}

	// DurationSecBuckets defines boundaries for second-scale operations
	// Covers spans from 1s to 1h with exponential growth
	// Useful for long-running computations, batch jobs
	DurationSecBuckets = []float64{
		1,    // 1s
		5,    // 5s
		15,   // 15s
		30,   // 30s
		60,   // 1m
		300,  // 5m
		900,  // 15m
		1800, // 30m
		3600, // 1h
	}

	// BytesBuckets defines boundaries for data size measurements
	// Covers spans from 1KB to 1GB with exponential growth
	// Useful for payload sizes, file operations
	BytesBuckets = []float64{
		1024,       // 1KB
		32768,      // 32KB
		1048576,    // 1MB
		33554432,   // 32MB
		134217728,  // 128MB
		536870912,  // 512MB
		1073741824, // 1GB
	}

	// CountBuckets defines boundaries for count-based measurements
	// Covers spans from 1 to 10000 with exponential growth
	// Useful for batch sizes, queue lengths, connection pools
	CountBuckets = []float64{
		1,
		10,
		50,
		100,
		500,
		1000,
		5000,
		10000,
	}
)
