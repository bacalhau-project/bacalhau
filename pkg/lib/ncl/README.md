# Bacalhau NCL (NATS Client Library)

## Overview

The NCL (NATS Client Library) is an internal library for Bacalhau, designed to provide reliable, scalable, and efficient communication between orchestrator and compute nodes. It leverages NATS for messaging and implements an event-driven architecture with support for asynchronous communication and granular event logging.

## Key Components

1. **Publisher**: Handles asynchronous message publishing with configurable subjects
2. **Ordered Publisher**: Extends Publisher with guaranteed message ordering and delivery tracking
3. **Subscriber**: Manages message consumption and processing
4. **MessageHandler**: Interface for processing received messages
5. **MessageFilter**: Interface for filtering incoming messages
6. **ProcessingNotifier**: Interface for receiving notifications of successfully processed messages
7. **Requester**: Handles request-response style messaging
8. **Responder**: Handles incoming requests and sends responses

## Technical Details

### Message Flow

1. **Publishing**:
   - Publishers accept messages through their `Publish` method
   - Messages are serialized using the [envelope](https://github.com/bacalhau-project/bacalhau/pkg/lib/envelope) library
   - Serialized messages are published to NATS using configured subjects
   - OrderedPublisher ensures messages are delivered in order and tracks delivery status

2. **Subscribing**:
   - Subscribers set up NATS subscriptions for specified subjects
   - When a message is received, it's deserialized using the envelope library
   - The message filter is applied to determine if it should be processed
   - Filtered messages are passed to configured MessageHandlers
   - Successfully processed messages trigger ProcessingNotifier callbacks

### Publisher Types

#### Basic Publisher
```go
// Synchronous publishing
Publish(ctx context.Context, request PublishRequest) error
```

#### Ordered Publisher
```go
// Asynchronous publishing with delivery tracking
PublishAsync(ctx context.Context, request PublishRequest) (PubFuture, error)

// Stream management
Reset(ctx context.Context)
Close(ctx context.Context) error
```

### Subscriber

The `subscriber` struct manages message consumption. Key features:

* NATS subscription management
* Message filtering
* Handler routing
* Processing notifications 
* Graceful shutdown

Key methods:
```go
Subscribe(subjects ...string) error
Close(ctx context.Context) error
```

### Request-Response Pattern

#### Requester
```go
// Send request and await response
Request(ctx context.Context, request PublishRequest) (*envelope.Message, error)
```

#### Responder
```go
// Listen for requests of specific type
Listen(ctx context.Context, messageType string, handler RequestHandler) error
```

## Usage Within Bacalhau

This library is designed to be used internally within the Bacalhau project. It should be integrated into the orchestrator and compute node components to handle all inter-node communication.

Example integration points:
1. Job assignment from orchestrator to compute nodes
2. Status updates from compute nodes to orchestrator
3. Heartbeat messages for health monitoring
4. Request-response interactions between components