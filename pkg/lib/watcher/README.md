# Watcher Library

## Overview

The Watcher Library is an internal component of the Bacalhau project that provides a robust event watching and processing system. It's designed to efficiently store, retrieve, and process events. The library ensures events are stored in a durable, ordered manner, allowing for consistent and reliable event processing. It supports features like checkpointing, filtering, and long-polling, while maintaining the ability to replay events from any point in the event history.


## Key Features

1. **Ordered Event Processing**: Events are processed in the exact order they were created, ensuring consistency and predictability in event handling.
2. **Durability**: Events are stored persistently in BoltDB, ensuring they survive system restarts or crashes.
3. **Replayability**: The system allows replaying events from any point in history, facilitating data recovery, debugging, and system reconciliation.
4. **Concurrency**: Multiple watchers can process events concurrently, improving system throughput.
5. **Filtering**: Watchers can filter events based on object types and operations, allowing for targeted event processing.
6. **Checkpointing**: Watchers can save their progress and resume from where they left off, enhancing reliability and efficiency.
7. **Long-polling**: Efficient event retrieval with support for long-polling, reducing unnecessary network traffic and database queries.
8. **Garbage Collection**: Automatic cleanup of old events to manage storage while maintaining the ability to replay from critical points.
9. **Flexible Event Iteration**: Different types of iterators for various use cases, including the ability to start from the oldest event, the latest event, or any specific point in the event history.


## Key Components

1. **Manager**: Manages multiple watchers and provides methods to create, lookup, and stop watchers.
2. **Watcher**: Represents a single event watcher that processes events sequentially.
3. **EventStore**: Responsible for storing and retrieving events, with BoltDB as the default implementation.
4. **EventHandler**: Interface for handling individual events.
5. **Serializer**: Handles the serialization and deserialization of events.

## Core Concepts

### Event

An `Event` represents a single occurrence in the system. It has the following properties:

- `SeqNum`: A unique, sequential identifier for the event.
- `Operation`: The type of operation (Create, Update, Delete).
- `ObjectType`: The type of object the event relates to.
- `Object`: The actual data associated with the event.
- `Timestamp`: When the event occurred.

### EventStore

The `EventStore` is responsible for persisting events and providing methods to retrieve them. It uses BoltDB as the underlying storage engine and supports features like caching, checkpointing, and garbage collection.

### Manager

The `Manager` manages multiple watchers and provides methods to create, lookup, and stop watchers.

### Watcher

A `Watcher` represents a single subscriber to events. It processes events sequentially and can be configured with filters and checkpoints.

### EventIterator

An `EventIterator` defines the starting position for reading events. There are four types of iterators:

1. **TrimHorizonIterator**: Starts from the oldest available event.
2. **LatestIterator**: Starts from the latest available event.
3. **AtSequenceNumberIterator**: Starts at a specific sequence number.
4. **AfterSequenceNumberIterator**: Starts after a specific sequence number.

## Usage

Here's how you typically use the Watcher library within Bacalhau:

1. Create an EventStore:

```go
db, _ := bbolt.Open("events.db", 0600, nil)
store, _ := boltdb.NewEventStore(db)
```

2. Create a manager:
```go
manager := watcher.NewManager(store)
```

3. Implement an EventHandler:
```go
type MyHandler struct{}

func (h *MyHandler) HandleEvent(ctx context.Context, event watcher.Event) error {
    // Process the event
    return nil
}
```


4. Create a watcher and set handler:

There are two main approaches to create and configure a watcher with a handler:

a.  Two-Step Creation (Handler After Creation):
```go
// Create watcher
w, _ := manager.Create(ctx, "my-watcher", 
    watcher.WithFilter(watcher.EventFilter{
        ObjectTypes: []string{"Job", "Execution"},
        Operations: []watcher.Operation{watcher.OperationCreate, watcher.OperationUpdate},
    }),
)

// Set handler
err = w.SetHandler(&MyHandler{})

// Start watching
err = w.Start(ctx)
```

b. One-Step Creation (With Auto-Start):
```go
w, _ := manager.Create(ctx, "my-watcher",
    watcher.WithHandler(&MyHandler{}),
    watcher.WithAutoStart(),
    watcher.WithFilter(watcher.EventFilter{
        ObjectTypes: []string{"Job", "Execution"},
        Operations: []watcher.Operation{watcher.OperationCreate, watcher.OperationUpdate},
    }),
)
```

5. Store events:
```go
store.StoreEvent(ctx, watcher.OperationCreate, "Job", jobData)
```


## Configuration

### Watch Configuration

When creating a watcher, you can configure it with various options:

- `WithInitialEventIterator(iterator EventIterator)`: Sets the starting position for watching if no checkpoint is found.
- `WithHandler(handler EventHandler)`: Sets the event handler for the watcher.
- `WithAutoStart()`: Enables automatic start of the watcher after creation.
- `WithFilter(filter EventFilter)`: Sets the event filter for watching.
- `WithBufferSize(size int)`: Sets the size of the event buffer.
- `WithBatchSize(size int)`: Sets the number of events to fetch in each batch.
- `WithInitialBackoff(backoff time.Duration)`: Sets the initial backoff duration for retries.
- `WithMaxBackoff(backoff time.Duration)`: Sets the maximum backoff duration for retries.
- `WithMaxRetries(maxRetries int)`: Sets the maximum number of retries for event handling.
- `WithRetryStrategy(strategy RetryStrategy)`: Sets the retry strategy for event handling.

Example:

```go
w, err := manager.Create(ctx, "my-watcher",
    watcher.WithInitialEventIterator(watcher.TrimHorizonIterator()),
    watcher.WithHandler(&MyHandler{}),
    watcher.WithAutoStart(),
    watcher.WithFilter(watcher.EventFilter{
        ObjectTypes: []string{"Job", "Execution"},
        Operations: []watcher.Operation{watcher.OperationCreate, watcher.OperationUpdate},
    }),
    watcher.WithBufferSize(1000),
    watcher.WithBatchSize(100),
    watcher.WithMaxRetries(3),
    watcher.WithRetryStrategy(watcher.RetryStrategyBlock),
)
```

### EventStore Configuration (BoltDB)

The BoltDB EventStore can be configured with various options:

- `WithEventsBucket(name string)`: Sets the name of the bucket used to store events.
- `WithCheckpointBucket(name string)`: Sets the name of the bucket used to store checkpoints.
- `WithEventSerializer(serializer watcher.Serializer)`: Sets the serializer used for events.
- `WithCacheSize(size int)`: Sets the size of the LRU cache used to store events.
- `WithLongPollingTimeout(timeout time.Duration)`: Sets the timeout duration for long-polling requests.
- `WithGCAgeThreshold(threshold time.Duration)`: Sets the age threshold for event pruning.
- `WithGCCadence(cadence time.Duration)`: Sets the interval at which garbage collection runs.
- `WithGCMaxRecordsPerRun(max int)`: Sets the maximum number of records to process in a single GC run.
- `WithGCMaxDuration(duration time.Duration)`: Sets the maximum duration for a single GC run.

Example:

```go
store, err := boltdb.NewEventStore(db,
    boltdb.WithEventsBucket("myEvents"),
    boltdb.WithCheckpointBucket("myCheckpoints"),
    boltdb.WithCacheSize(1000),
    boltdb.WithLongPollingTimeout(10*time.Second),
)
```


## Best Practices

1. Use meaningful watcher IDs to easily identify different components subscribing to events.
2. Implement error handling in your `EventHandler` to ensure robust event processing.
3. Use appropriate filters to minimize unnecessary event processing.
4. Regularly checkpoint your watchers to enable efficient restarts.
5. Monitor watcher stats to ensure they're keeping up with event volume.

## Troubleshooting

1. If a watcher is falling behind, consider increasing the batch size or optimizing the event handling logic.
2. For performance issues, check the BoltDB file size and consider tuning the garbage collection parameters.


## Future Improvements
1. Enhanced monitoring and metrics.