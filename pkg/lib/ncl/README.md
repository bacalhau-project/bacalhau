# Bacalhau NCL (NATS Client Library)

## Overview

The NCL (NATS Client Library) is an internal library for Bacalhau, designed to provide reliable, scalable, and efficient communication between orchestrator and compute nodes. It leverages NATS for messaging and implements an event-driven architecture with support for both publish-subscribe and request-response patterns.

## Key Components

1. **Publisher**: Handles all message delivery patterns
   - Synchronous publishing with delivery guarantees
   - Request-response communication
   - Optional message ordering and delivery tracking
   - Configurable subjects and message routing

2. **Subscriber**: Manages message consumption
   - Subject-based subscription management
   - Message filtering capabilities
   - Automatic acknowledgments and retries
   - Processing notifications
   - Backoff strategies for failures

3. **Responder**: Handles request-response pattern
   - Type-based request routing
   - Handler registration
   - Automatic error responses
   - Request timeouts

4. **Encoder**: Manages message serialization
   - Consistent message encoding/decoding
   - Metadata enrichment
   - Error handling
   - Type registration

## Technical Details

### Message Flow

1. **Publishing and Requesting**:
   - Messages are encoded using the encoder
   - Messages are published to configured subjects
   - Optional request-response with reply subjects
   - OrderedPublisher ensures delivery ordering

2. **Subscribing**:
   - Subscribers set up NATS subscriptions
   - Messages are decoded using the encoder
   - Filters determine message processing
   - Handlers process valid messages
   - Success/failure notifications are sent
   - Automatic ack/nack with backoff

### Component Interfaces

#### Publisher
```go
type Publisher interface {
    // Fire-and-forget publishing
    Publish(ctx context.Context, request PublishRequest) error
    
    // Request-response pattern
    Request(ctx context.Context, request PublishRequest) (*envelope.Message, error)
}
```

#### OrderedPublisher
```go
type OrderedPublisher interface {
    Publisher
    PublishAsync(ctx context.Context, request PublishRequest) (PubFuture, error)
    Reset(ctx context.Context)
    Close(ctx context.Context) error
}
```

#### Subscriber
```go
type Subscriber interface {
    Subscribe(ctx context.Context, subjects ...string) error
    Close(ctx context.Context) error
}
```

#### Responder
```go
type Responder interface {
    Listen(ctx context.Context, messageType string, handler RequestHandler) error
    Close(ctx context.Context) error
}
```

### Example Usage

```go
// Create a publisher for both publishing and requests
publisher, _ := NewPublisher(nc, PublisherConfig{
    Name:            "compute-node",
    MessageRegistry: registry,
})

// Publish a message
err := publisher.Publish(ctx, NewPublishRequest(message))

// Make a request
response, err := publisher.Request(ctx, NewPublishRequest(request))

// Create a responder for handling requests
responder, _ := NewResponder(nc, ResponderConfig{
    Name:     "orchestrator",
    Subject:  "requests",
})

// Register request handlers
err = responder.Listen(ctx, "JobRequest", handler)

// Create a subscriber for message consumption
subscriber, _ := NewSubscriber(nc, SubscriberConfig{
    Name:           "worker",
    MessageHandler: handler,
})

// Subscribe to subjects
err = subscriber.Subscribe(ctx, "updates.>")
```

## Usage Within Bacalhau

This library is designed to be used internally within the Bacalhau project. It integrates into the orchestrator and compute node components to handle all inter-node communication.

Example integration points:
1. Job assignment from orchestrator to compute nodes
2. Status updates from compute nodes to orchestrator
3. Heartbeat messages for health monitoring
4. Compute node registration and discovery

Each component type (Publisher, Subscriber, Responder) handles specific communication patterns while sharing the same underlying message encoding and metadata handling through the encoder.