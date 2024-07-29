# Bacalhau NCL (NATS Client Library)

## Overview

The NCL (NATS Client Library) is an internal library for Bacalhau, designed to provide reliable, scalable, and efficient communication between orchestrator and compute nodes. It leverages NATS for messaging and implements an event-driven architecture with support for asynchronous communication, granular event logging, and robust state management.

## Key Components

1. **EnvelopedRawMessageSerDe**: Handles serialization and deserialization of RawMessages with versioning and CRC checks.
2. **MessageSerDeRegistry**: Manages serialization and deserialization of different message types.
3. **Publisher**: Handles asynchronous message publishing.
4. **Subscriber**: Manages message consumption and processing.
5. **MessageHandler**: Interface for processing received messages.
6. **MessageFilter**: Interface for filtering incoming messages.
7. **Checkpointer**: Interface for managing checkpoints in message processing.

## Technical Details

### Message Flow

1. **Publishing**:
   - The publisher accepts a `Message` struct through its `Publish` method.
   - The `MessageSerDeRegistry` serializes the `Message` into a `RawMessage` using the appropriate `MessageSerDe` for the message type.
   - The `EnvelopedRawMessageSerDe` serializes the `RawMessage` into a byte slice with an envelope containing a version byte and a CRC checksum.
   - The serialized message is published to NATS using the configured subject.

2. **Subscribing**:
   - The subscriber sets up a NATS subscription for specified subjects.
   - When a message is received, it's passed to the `processMessage` method.
   - The `EnvelopedRawMessageSerDe` deserializes the raw bytes into a `RawMessage`. The envelope version helps determine the deserialization method, and the CRC checksum is used to verify the message integrity.
   - The message filter is applied to determine if the message should be processed.
   - The `MessageSerDeRegistry` deserializes the `RawMessage` into a `Message` using the appropriate `MessageSerDe` for the message type.
   - The deserialized `Message` is passed to each configured `MessageHandler`.

### Serialization/Deserialization (SerDe) Flow

1. **Message to bytes (for sending)**:
   `Message` -> `MessageSerDe.Serialize()` -> `RawMessage` -> `EnvelopedRawMessageSerDe.Serialize()` -> `[]byte`

2. **Bytes to Message (when receiving)**:
   `[]byte` -> `EnvelopedRawMessageSerDe.Deserialize()` -> `RawMessage` -> `MessageSerDe.Deserialize()` -> `Message`

### EnvelopedRawMessageSerDe

The `EnvelopedRawMessageSerDe` adds a version byte and a CRC checksum to each serialized `RawMessage`. The envelope structure is as follows:

```
+----------------+----------------+--------------------+
| Version (1 byte)| CRC (4 bytes) | Serialized Message |
+----------------+----------------+--------------------+
```

This allows for future extensibility, backward compatibility, and data integrity verification.

### MessageSerDeRegistry

The `MessageSerDeRegistry` manages the serialization and deserialization of different message types. It allows registering custom message types with unique names and provides methods for serializing and deserializing messages.

Key methods:
- `Register(name string, messageType any, serde MessageSerDe) error`
- `Serialize(message *Message) (*RawMessage, error)`
- `Deserialize(rawMessage *RawMessage) (*Message, error)`

### Publisher

The `publisher` struct handles message publishing. It supports asynchronous publishing and can be configured with options like message serializer, MessageSerDeRegistry, and destination subject or prefix.

Key method:
- `Publish(ctx context.Context, message *Message) error`

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