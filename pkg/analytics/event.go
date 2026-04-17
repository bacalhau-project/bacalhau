package analytics

const (
	// NoopEventType is a special event type marker used for no-operation events.
	// Events with this type are automatically filtered out by the Emit function,
	// ensuring they won't be sent to the analytics backend.
	NoopEventType = "bacalhau.noop"
)

// EventProperties is a type alias for the property map used in events
// to improve code readability and type clarity.
type EventProperties map[string]interface{}

// Event represents a analytics event that can be sent to the analytics backend.
// Events have a type (name) and a set of properties.
type Event interface {
	// Type returns the type (name) of the event.
	// This should be a dot-separated string like "bacalhau.job_v1.submit".
	Type() string

	// Properties returns the properties of the event.
	// These are structured data fields that provide details about the event.
	Properties() EventProperties
}

// baseEvent is a lightweight, pre-computed event implementation
// that avoids reflection for maximum efficiency.
// It stores the event type and properties directly, allowing them
// to be computed once and reused.
type baseEvent struct {
	eventType string
	props     EventProperties
}

// NewEvent creates a new baseEvent with the given type and properties.
// This is the recommended way to create events for most use cases.
//
// Parameters:
//   - eventType: The type (name) of the event (e.g., "bacalhau.job_v1.submit")
//   - props: A map of properties that describe the event
//
// Returns a baseEvent that implements the Event interface.
func NewEvent(eventType string, props EventProperties) Event {
	return &baseEvent{
		eventType: eventType,
		props:     props,
	}
}

// Type returns the type of the event.
// Implements the Event interface.
func (e *baseEvent) Type() string {
	return e.eventType
}

// Properties returns the pre-computed properties map.
// Implements the Event interface.
//
// Since baseEvent stores properties directly, this is very efficient
// as it simply returns the stored reference without computation.
func (e *baseEvent) Properties() EventProperties {
	return e.props
}

// noopEvent is a specialized implementation of the Event interface
// that always returns the same fixed values and is always ignored
// by the analytics system.
type noopEvent struct{}

// NoopEvent is a pre-initialized singleton instance of noopEvent.
// Use this singleton directly when you need to emit an event
// that should be discarded by the analytics system.
//
// Example usage:
//
//	analytics.Emit(analytics.NoopEvent) // Will be ignored by the analytics system
var NoopEvent Event = &noopEvent{}

// Type returns the NoopEventType constant.
// This value is recognized by the Emit function to filter out this event.
func (e *noopEvent) Type() string {
	return NoopEventType
}

// Properties returns an empty properties map.
// Since noopEvent instances are filtered out by Emit based on their Type,
// this method is required to satisfy the Event interface but is not used
// when the event is processed.
func (e *noopEvent) Properties() EventProperties {
	return EventProperties{}
}
