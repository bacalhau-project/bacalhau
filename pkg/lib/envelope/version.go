package envelope

import (
	"fmt"
)

// Version and size constants
const (
	SchemaVersionJSONV1     SchemaVersion = 1
	SchemaVersionProtobufV1 SchemaVersion = 2
	DefaultSchemaVersion                  = SchemaVersionJSONV1
)

// SchemaVersion represents the version of the serialization schema
type SchemaVersion byte

// String returns a string representation of the schema version
func (v SchemaVersion) String() string {
	switch v {
	case SchemaVersionJSONV1:
		return "json-v1"
	case SchemaVersionProtobufV1:
		return "protobuf-v1"
	default:
		return fmt.Sprintf("0x%02x", byte(v))
	}
}
