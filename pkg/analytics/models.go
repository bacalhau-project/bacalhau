package analytics

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	otellog "go.opentelemetry.io/otel/log"
)

type EventType string

type Event struct {
	Type       string
	Properties any
}

// NewEvent creates a new Event.
func NewEvent(eventType string, properties any) *Event {
	return &Event{
		Type:       eventType,
		Properties: properties,
	}
}

// ToLogRecord converts an Event to a LogRecord.
func (e *Event) ToLogRecord() (otellog.Record, error) {
	// Convert the event properties to json
	propertiesJSON, err := json.Marshal(e.Properties)
	if err != nil {
		return otellog.Record{}, err
	}

	record := otellog.Record{}
	record.AddAttributes(
		otellog.String("event", e.Type),
		otellog.String("properties", string(propertiesJSON)),
	)
	record.SetTimestamp(time.Now().UTC())
	record.SetBody(otellog.StringValue(""))
	return record, nil
}

func hashString(in string) string {
	hash := sha256.New()
	hash.Write([]byte(in))
	return hex.EncodeToString(hash.Sum(nil))
}
