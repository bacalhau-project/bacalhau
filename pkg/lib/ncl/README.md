# Bacalhau NCL (NATS Client Library)

## Overview

The NCL (NATS Client Library) is an internal library for Bacalhau, designed to provide reliable, scalable, and efficient communication between orchestrator and compute nodes. It leverages NATS for messaging and implements an event-driven architecture with support for asynchronous communication, granular event logging, and robust state management.

## Key Components

1. **EnvelopeSerializer**: Handles message enveloping with versioning and CRC checks.
2. **PayloadRegistry**: Manages serialization and deserialization of different payload types.
3. **Publisher**: Handles asynchronous message publishing.
4. **Subscriber**: Manages message consumption and processing.
5. **MessageHandler**: Interface for processing received messages.
6. **MessageFilter**: Interface for filtering incoming messages.
7. **Checkpointer**: Interface for managing checkpoints in message processing.

## Technical Details

### Message Flow

1. **Publishing**:
    - The publisher accepts any type of payload through its `Publish` or `PublishWithMetadata` methods.
    - The payload is serialized using `PayloadRegistry.SerializePayload()`, with json as the current supported format.
    - The serialized payload is wrapped in a `RawMessage` struct along with metadata.
    - The `EnvelopeSerializer` serializes the `RawMessage` into a byte slice with an envelope containing a version byte and a CRC checksum. The `RawMessage` is serialized using either json or protobuf.
    - The serialized message is published to NATS using the configured subject.

2. **Subscribing**:
    - The subscriber sets up a NATS subscription for specified subjects.
    - When a message is received, it's passed to the `processMessage` method.
    - The `EnvelopeSerializer` deserializes the raw bytes into a `RawMessage`. The envelope version helps determine the deserialization method , json or protobuf, and the CRC checksum is used to verify the message integrity.
    - The message filter is applied to determine if the message should be processed.
    - The payload is deserialized using `PayloadRegistry.DeserializePayload()`.
    - The deserialized message is passed to each configured `MessageHandler`.

### EnvelopeSerializer

The `EnvelopeSerializer` adds a version byte and a CRC checksum to each serialized message. The envelope structure is as follows:

```
+----------------+----------------+--------------------+
| Version (1 byte)| CRC (4 bytes) | Serialized Message |
+----------------+----------------+--------------------+
```

This allows for future extensibility, backward compatibility, and data integrity verification.

### PayloadRegistry

The `PayloadRegistry` manages the serialization and deserialization of different payload types. It allows registering custom payload types with unique names and provides methods for serializing and deserializing payloads.

Key methods:
- `Register(name string, payload any) error`
- `SerializePayload(metadata *Metadata, payload any) ([]byte, error)`
- `DeserializePayload(metadata *Metadata, data []byte) (any, error)`

### Publisher

The `publisher` struct handles message publishing. It supports asynchronous publishing and can be configured with options like message serializer, payload registry, and destination subject or prefix.

Key methods:
- `Publish(ctx context.Context, event any) error`
- `PublishWithMetadata(ctx context.Context, metadata *Metadata, event any) error`

### Subscriber

The `subscriber` struct manages message consumption. It sets up NATS subscriptions, processes incoming messages, and routes them to the appropriate message handlers.

Key methods:
- `Subscribe(subjects ...string) error`
- `Close(ctx context.Context) error`

## Usage Within Bacalhau

This library is designed to be used internally within the Bacalhau project. It should be integrated into the orchestrator and compute node components to handle all inter-node communication.

Example integration points:
1. Job assignment from orchestrator to compute nodes
2. Status updates from compute nodes to orchestrator
3. Heartbeat messages for health monitoring
