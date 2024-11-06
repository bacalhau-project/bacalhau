package envelope

import (
	"fmt"
)

const (
	ErrNilMessage = "message is nil"
	ErrEmptyData  = "data is empty"
)

// ErrUnsupportedEncoding is returned when an unsupported encoding is encountered.
type ErrUnsupportedEncoding struct {
	Encoding string
}

// NewErrUnsupportedEncoding creates a new ErrUnsupportedEncoding error.
func NewErrUnsupportedEncoding(encoding string) *ErrUnsupportedEncoding {
	return &ErrUnsupportedEncoding{Encoding: encoding}
}

// Error implements the error interface for ErrUnsupportedEncoding.
func (e *ErrUnsupportedEncoding) Error() string {
	return fmt.Sprintf("unsupported encoding: %s", e.Encoding)
}

// ErrUnsupportedMessageType is returned when an unsupported message type is encountered.
type ErrUnsupportedMessageType struct {
	Type string
}

// NewErrUnsupportedMessageType creates a new ErrUnsupportedMessageType error.
func NewErrUnsupportedMessageType(messageType string) *ErrUnsupportedMessageType {
	return &ErrUnsupportedMessageType{Type: messageType}
}

// Error implements the error interface for ErrUnsupportedMessageType.
func (e *ErrUnsupportedMessageType) Error() string {
	return fmt.Sprintf("unsupported message type: %s", e.Type)
}

// ErrBadMessage is returned when a message is malformed or invalid.
type ErrBadMessage struct {
	Reason string
}

// NewErrBadMessage creates a new ErrBadMessage error.
func NewErrBadMessage(reason string) *ErrBadMessage {
	return &ErrBadMessage{Reason: reason}
}

// Error implements the error interface for ErrBadMessage.
func (e *ErrBadMessage) Error() string {
	return fmt.Sprintf("bad message: %s", e.Reason)
}

// ErrBadPayload is returned when a payload is malformed or invalid.
type ErrBadPayload struct {
	Reason string
}

// NewErrBadPayload creates a new ErrBadPayload error.
func NewErrBadPayload(reason string) *ErrBadPayload {
	return &ErrBadPayload{Reason: reason}
}

// Error implements the error interface for ErrBadPayload.
func (e *ErrBadPayload) Error() string {
	return fmt.Sprintf("bad payload: %s", e.Reason)
}

// ErrSerializationFailed is returned when serialization fails.
type ErrSerializationFailed struct {
	Encoding string
	Err      error
}

// NewErrSerializationFailed creates a new ErrSerializationFailed error.
func NewErrSerializationFailed(encoding string, err error) error {
	return &ErrSerializationFailed{Encoding: encoding, Err: err}
}

// Error implements the error interface for ErrSerializationFailed.
func (e *ErrSerializationFailed) Error() string {
	return fmt.Sprintf("failed to serialize with encoding %s: %v", e.Encoding, e.Err)
}

// Unwrap returns the underlying error for ErrSerializationFailed.
func (e *ErrSerializationFailed) Unwrap() error {
	return e.Err
}

// ErrDeserializationFailed is returned when payload deserialization fails.
type ErrDeserializationFailed struct {
	Encoding string
	Err      error
}

// NewErrDeserializationFailed creates a new ErrDeserializationFailed error.
func NewErrDeserializationFailed(encoding string, err error) error {
	return &ErrDeserializationFailed{Encoding: encoding, Err: err}
}

// Error implements the error interface for ErrDeserializationFailed.
func (e *ErrDeserializationFailed) Error() string {
	return fmt.Sprintf("failed to deserialize with encoding %s: %v", e.Encoding, e.Err)
}

// Unwrap returns the underlying error for ErrDeserializationFailed.
func (e *ErrDeserializationFailed) Unwrap() error {
	return e.Err
}

// ErrUnexpectedPayloadType is returned when the payload type is unexpected.
type ErrUnexpectedPayloadType struct {
	Expected string
	Actual   string
}

// NewErrUnexpectedPayloadType creates a new ErrUnexpectedPayloadType error.
func NewErrUnexpectedPayloadType(expected, actual string) error {
	return &ErrUnexpectedPayloadType{Expected: expected, Actual: actual}
}

// Error implements the error interface for ErrUnexpectedPayloadType.
func (e *ErrUnexpectedPayloadType) Error() string {
	return fmt.Sprintf("unexpected payload type: expected %s, got %s", e.Expected, e.Actual)
}

// Ensure all custom error types implement the error interface.
var (
	_ error = (*ErrUnsupportedEncoding)(nil)
	_ error = (*ErrUnsupportedMessageType)(nil)
	_ error = (*ErrBadMessage)(nil)
	_ error = (*ErrBadPayload)(nil)
	_ error = (*ErrSerializationFailed)(nil)
	_ error = (*ErrDeserializationFailed)(nil)
	_ error = (*ErrUnexpectedPayloadType)(nil)
)

// Ensure error types that wrap other errors implement the unwrap interface.
var (
	_ interface{ Unwrap() error } = (*ErrSerializationFailed)(nil)
	_ interface{ Unwrap() error } = (*ErrDeserializationFailed)(nil)
)
