package watcher

import (
	"strconv"
)

// EventIteratorType defines the type of starting position for reading events.
type EventIteratorType int

const (
	// EventIteratorTrimHorizon specifies that events should be read from the oldest available event.
	EventIteratorTrimHorizon EventIteratorType = iota
	// EventIteratorLatest specifies that events should be read from the latest available event.
	EventIteratorLatest
	// EventIteratorAtSequenceNumber specifies that events should be read starting from a specific sequence number.
	EventIteratorAtSequenceNumber
	// EventIteratorAfterSequenceNumber specifies that events should be read starting after a specific sequence number.
	EventIteratorAfterSequenceNumber
)

// EventIterator defines the starting position for reading events.
type EventIterator struct {
	Type           EventIteratorType
	SequenceNumber uint64
}

// TrimHorizonIterator creates an EventIterator that starts from the oldest available event.
func TrimHorizonIterator() EventIterator {
	return EventIterator{
		Type: EventIteratorTrimHorizon,
	}
}

// LatestIterator creates an EventIterator that starts from the latest available event.
func LatestIterator() EventIterator {
	return EventIterator{
		Type: EventIteratorLatest,
	}
}

// AtSequenceNumberIterator creates an EventIterator that starts at a specific sequence number.
func AtSequenceNumberIterator(seqNum uint64) EventIterator {
	return EventIterator{
		Type:           EventIteratorAtSequenceNumber,
		SequenceNumber: seqNum,
	}
}

// AfterSequenceNumberIterator creates an EventIterator that starts after a specific sequence number.
func AfterSequenceNumberIterator(seqNum uint64) EventIterator {
	return EventIterator{
		Type:           EventIteratorAfterSequenceNumber,
		SequenceNumber: seqNum,
	}
}

// String returns a string representation of the EventIterator.
func (sp EventIterator) String() string {
	switch sp.Type {
	case EventIteratorTrimHorizon:
		return "trim_horizon"
	case EventIteratorLatest:
		return "latest"
	case EventIteratorAtSequenceNumber:
		return "at_sequence_number(" + strconv.FormatUint(sp.SequenceNumber, 10) + ")"
	case EventIteratorAfterSequenceNumber:
		return "after_sequence_number(" + strconv.FormatUint(sp.SequenceNumber, 10) + ")"
	default:
		return "unknown"
	}
}
