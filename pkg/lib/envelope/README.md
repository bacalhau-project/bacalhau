# Envelope Library

A serialization library that provides versioned, type-safe message handling with integrity checks.

## Overview

The envelope library wraps messages with version information and CRC checksums, enabling:

- Type-safe message handling with compile-time checks
- Multiple serialization formats (JSON, Protocol Buffers)
- Version-based backward compatibility
- Data integrity verification
- Flexible metadata support

## Key Components

### Message & Metadata

The core message types that applications work with:

```go
// Create a message with payload
msg := envelope.NewMessage(MyPayload{...})
msg.WithMetadataValue("key", "value")
```

### Registry

Manages payload type registration and serialization:
```go
// Create registry and register types
registry := envelope.NewRegistry()
registry.Register("MyPayload", MyPayload{})

// Serialize message
encoded, err := registry.Serialize(msg)

// Deserialize message
decoded, err := registry.Deserialize(encoded)
```

### Serializer
Handles low-level message serialization with versioning and CRC checks:
```go
// Create serializer with default JSON/Protobuf support
serializer := envelope.NewSerializer()

// Serialize with envelope
data, err := serializer.Serialize(encoded)

// Deserialize and verify
result, err := serializer.Deserialize(data)
```

## Message Format
Messages are wrapped in an envelope structure:
```
+----------------+----------------+--------------------+
| Version (1 byte)| CRC (4 bytes) | Serialized Message |
+----------------+----------------+--------------------+
```

* Version: Indicates serialization format/version
* CRC: 32-bit checksum for data integrity
* Message: Serialized message content

## Serialization Formats

* JSON (default)
* Protocol Buffers v1
* Custom formats can be added by implementing MessageSerializer interface

## Usage Example
```go
// 1. Set up registry
registry := envelope.NewRegistry()
registry.Register("MyPayload", MyPayload{})

// 2. Create message
msg := envelope.NewMessage(MyPayload{
Field: "value",
})
msg.WithMetadataValue("version", "1.0")

// 3. Serialize for transmission
encoded, err := registry.Serialize(msg)
if err != nil {
// Handle error
}

// 4. Deserialize received message
decoded, err := registry.Deserialize(encoded)
if err != nil {
// Handle error
}

// 5. Type-safe payload access
payload, ok := decoded.GetPayload(MyPayload{})
```

## Best Practices
* Register all payload types at startup
* Use strongly typed payloads instead of maps/interfaces
* Include version info in metadata for schema evolution
* Verify CRC matches before processing messages