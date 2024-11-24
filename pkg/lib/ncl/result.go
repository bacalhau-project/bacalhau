package ncl

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	// statusHdr is the header key that nats uses to indicate the status of the message
	statusHdr = "Status"

	// noResponders is the value of the status header when no responders are available
	noResponders = "503"
)

// Result represents the result of message processing
type Result struct {
	Error string `json:"error,omitempty"`
	Delay int64  `json:"delay,omitempty"`
}

// NewResult creates a new reply
func NewResult() *Result {
	return &Result{}
}

// WithError sets the error message
func (r *Result) WithError(err error) *Result {
	if err != nil {
		r.Error = err.Error()
	}
	return r
}

// WithDelay sets the delay
func (r *Result) WithDelay(delay time.Duration) *Result {
	r.Delay = delay.Nanoseconds()
	return r
}

// Err converts result to error if it represents a failure
func (r *Result) Err() error {
	if r.Error != "" {
		return fmt.Errorf(r.Error)
	}
	return nil
}

// DelayDuration converts delay to time.Duration
func (r *Result) DelayDuration() time.Duration {
	return time.Duration(r.Delay)
}

// Ack creates a success result and publishes it
func Ack(m *nats.Msg) error {
	return sendResult(m, NewResult())
}

// Nack creates an error result
func Nack(m *nats.Msg, err error) error {
	return sendResult(m, NewResult().WithError(err))
}

func NackWithDelay(m *nats.Msg, err error, delay time.Duration) error {
	return sendResult(m, NewResult().WithError(err).WithDelay(delay))
}

// Handle serialization in one place
func sendResult(m *nats.Msg, resp *Result) error {
	if m == nil {
		return fmt.Errorf("message cannot be nil")
	}
	if m.Reply == "" {
		// No reply subject, nothing to do
		return nil
	}
	replyType := "ack"
	if resp.Error != "" {
		replyType = "nack"
	}
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal %s result: %w", replyType, err)
	}
	if err = m.Respond(data); err != nil {
		return fmt.Errorf("failed to send %s result: %w", replyType, err)
	}
	return nil
}

// ParseResult parses result from NATS message
func ParseResult(msg *nats.Msg) (*Result, error) {
	if len(msg.Data) == 0 {
		if msg.Header.Get(statusHdr) == noResponders {
			return nil, nats.ErrNoResponders
		}
		return nil, fmt.Errorf("empty message data with headers: %v", msg.Header)
	}
	result := new(Result)
	if err := json.Unmarshal(msg.Data, result); err != nil {
		return nil, fmt.Errorf("failed to parse result: %w", err)
	}
	return result, nil
}
