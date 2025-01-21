# MetricRecorder

MetricRecorder is a helper for recording OpenTelemetry metrics with consistent attribute handling and aggregation capabilities. It simplifies the process of recording latencies, counters, and gauges while maintaining a clean API.

## Features

- Aggregates metrics internally until explicitly published
  - Perfect for loops - automatically sums up latencies for each sub-operation type
  - Reduces number of metrics published to just the totals
- Records operation latencies with sub-operation tracking
- Supports counters with increment-by-one and increment-by-n operations
- Handles gauge measurements
- Manages attributes consistently across all measurements
- Built-in error type recording using OpenTelemetry semantic conventions
- Not thread-safe by design (one recorder per goroutine)

## Usage

### Basic Usage

```go
// Create a new recorder with base attributes
recorder := NewMetricRecorder(attribute.String("operation", "process"))
// Ensure metrics are published when done
defer recorder.Done(ctx, totalDurationHistogram)

// Record latency for specific operations
recorder.Latency(ctx, dequeueHistogram, "dequeue")

// Count operation
recorder.Count(ctx, operationCounter)

// Record gauge values
recorder.Gauge(ctx, queueSizeGauge, float64(queueSize))
```

### Tracking Sub-Operations

```go
func ProcessJob(ctx context.Context, job *Job) (err error) {
    // Create recorder with base attributes
    recorder := NewMetricRecorder(
        attribute.String("job_type", job.Type),
        attribute.String("priority", job.Priority),
    )
    // Records total duration when done
    defer recorder.Done(ctx, jobTotalDurationHist)
    defer recorder.Error(err)

    // Each Latency() call measures time since the previous operation
    if err := validateJob(job); err != nil {
        return err
    }
    recorder.Latency(ctx, jobStepHist, "validation")

    if err := processJob(job); err != nil {
        return err
    }
    recorder.Latency(ctx, jobStepHist, "processing")

    cleanup(job)
    recorder.Latency(ctx, jobStepHist, "cleanup")

    return nil
}
```

### Aggregating Metrics in Loops

```go
func ProcessBatch(ctx context.Context, items []Item) (err error) {
    recorder := NewMetricRecorder(attribute.String("operation", "batch_process"))
    defer recorder.Done(ctx, batchDurationHist)
    defer recorder.Error(err)

    for _, item := range items {
        // These latencies are automatically summed by operation type
        if err := validate(item); err != nil {
            return
        }
        recorder.Latency(ctx, stepHist, "validation")
    
        if err := unmarshall(item); err != nil {
            return
        }
        recorder.Latency(ctx, stepHist, "unmarshalling")
    
        if err := process(item); err != nil {
            return
        }
        recorder.Latency(ctx, stepHist, "processing")
    }
    // When Done() is called:
    // - "validation" latency will be the total time spent in validation across all items
    // - "unmarshalling" latency will be the total time spent in unmarshalling across all items
    // - "processing" latency will be the total time spent in processing across all items
    return nil
}
```

### Recording Errors

```go
if err := process(msg); err != nil {
    // Records error type using OpenTelemetry semantic conventions
    recorder.Error(err)
    return err
}
```

### Adding Attributes

```go
// Add attributes at creation
recorder := NewMetricRecorder(
    attribute.String("service", "processor"),
    attribute.String("version", "1.0"),
)

// Add attributes later
recorder.AddAttributes(attribute.Int("retry_count", retryCount))
```

### Recording Different Metric Types

```go
// Record latency since last operation
recorder.Latency(ctx, processHistogram, "process")

// Increment counter by 1
recorder.Count(ctx, requestCounter)

// Increment counter by specific value
recorder.CountN(ctx, bytesProcessedCounter, bytesProcessed)

// Set gauge value
recorder.Gauge(ctx, activeWorkersGauge, float64(workerCount))

// Record specific duration
recorder.Duration(ctx, customDurationHist, measureDuration)
```

## Important Notes

1. **Thread Safety**: MetricRecorder is not thread-safe. Create separate recorders for each goroutine if you need to record metrics from multiple goroutines.

2. **Lifecycle Management**:
    - The recorder starts timing when created
    - Metrics are aggregated internally until `Done()` is called
    - Call `Done()` to publish all aggregated metrics
    - Use `defer recorder.Done(ctx, histogram)` right after creation

3. **Attribute Handling**:
    - Base attributes are set at creation
    - Additional attributes can be added later
    - All attributes are included in every metric recording
    - Final attributes can be added when calling `Done()`

4. **Aggregation Behavior**:
    - Latencies and counts are aggregated internally
    - Gauges and direct durations are published immediately
    - All aggregated metrics are published when `Done()` is called