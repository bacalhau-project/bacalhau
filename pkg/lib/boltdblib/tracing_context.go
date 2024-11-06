package boltdblib

import (
	"time"

	"github.com/rs/zerolog/log"
)

const defaultTracingDuration = 10 * time.Millisecond

// TracingContext is a context that can be used to trace the duration of a transaction
// and log a debug message if it exceeds a certain threshold.
type TracingContext struct {
	TxContext
	startTime       time.Time
	committed       bool
	tracingDuration time.Duration
}

// NewTracingContext creates a new tracing context
func NewTracingContext(ctx TxContext) *TracingContext {
	return &TracingContext{
		TxContext:       ctx,
		startTime:       time.Now(),
		tracingDuration: defaultTracingDuration,
	}
}

// WithTracingDuration sets the duration threshold for transaction tracing
func (t *TracingContext) WithTracingDuration(d time.Duration) *TracingContext {
	t.tracingDuration = d
	return t
}

func (t *TracingContext) Commit() error {
	t.logIfSlow("commit")
	err := t.TxContext.Commit()
	if err == nil {
		t.committed = true
	}
	return err
}

func (t *TracingContext) Rollback() error {
	if !t.committed {
		t.logIfSlow("rollback")
	}
	return t.TxContext.Rollback()
}

// logIfSlow logs a debug message if the duration exceeds the threshold
func (t *TracingContext) logIfSlow(action string) {
	elapsed := time.Since(t.startTime)
	if elapsed > t.tracingDuration {
		log.Debug().Msgf("transaction took %s to %s", elapsed.String(), action)
	}
}

// compile time check whether the TracingContext implements the TxContext interface
var _ TxContext = (*TracingContext)(nil)
