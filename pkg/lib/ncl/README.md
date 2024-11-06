# Bacalhau NCL (NATS Client Library)

## Overview

The NCL (NATS Client Library) is an internal library for Bacalhau, designed to provide reliable, scalable, and efficient communication between orchestrator and compute nodes. It leverages NATS for messaging and implements an event-driven architecture with support for asynchronous communication and granular event logging.

## Key Components

1. **Publisher**: Handles asynchronous message publishing
2. **Subscriber**: Manages message consumption and processing
3. **MessageHandler**: Interface for processing received messages
4. **MessageFilter**: Interface for filtering incoming messages
5. **Checkpointer**: Interface for managing checkpoints in message processing

## Technical Details

### Message Flow

1. **Publishing**:
   - The publisher accepts a message through its `Publish` method
   - Message is serialized using the [envelope](https://github.com/bacalhau-project/bacalhau/pkg/lib/envelope) library
   - The serialized message is published to NATS using the configured subject

2. **Subscribing**:
   - The subscriber sets up a NATS subscription for specified subjects
   - When a message is received, it's deserialized using the envelope library
   - The message filter is applied to determine if it should be processed
   - Filtered messages are passed to configured MessageHandlers


### Publisher

The `publisher` struct handles message publishing. It supports:

- Asynchronous publishing
- Configurable destination subjects/prefixes
- Message retries and timeout handling
- Error handling and recovery

Key method:
```go
Publish(ctx context.Context, message *Message) error
```

### Subscriber

The `subscriber` struct manages message consumption. Key features:

* NATS subscription management
* Message filtering
* Handler routing
* Checkpoint management
* Graceful shutdown

Key methods:
```go
Subscribe(subjects ...string) error
Close(ctx context.Context) error
```

## Usage Within Bacalhau

This library is designed to be used internally within the Bacalhau project. It should be integrated into the orchestrator and compute node components to handle all inter-node communication.

Example integration points:
1. Job assignment from orchestrator to compute nodes
2. Status updates from compute nodes to orchestrator
3. Heartbeat messages for health monitoring