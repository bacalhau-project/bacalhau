package jobstore

import (
	"time"

	"github.com/rs/zerolog/log"
)

const maxTracingDuration = 10 * time.Millisecond

// TracingContext is a context that can be used to trace the duration of a transaction
// and log a debug message if it exceeds a certain threshold.
type TracingContext struct {
	TxContext
	startTime time.Time
}

// NewTracingContext creates a new tracing context
func NewTracingContext(ctx TxContext) *TracingContext {
	return &TracingContext{
		TxContext: ctx,
		startTime: time.Now(),
	}
}

func (t TracingContext) Commit() error {
	t.logIfSlow("commit")
	return t.TxContext.Commit()
}

func (t TracingContext) Rollback() error {
	t.logIfSlow("rollback")
	return t.TxContext.Rollback()
}

// logIfSlow logs a debug message if the duration exceeds the threshold
func (t TracingContext) logIfSlow(action string) {
	elapsed := time.Since(t.startTime)
	if elapsed > maxTracingDuration {
		log.Debug().Msgf("transaction took %s to %s", elapsed.String(), action)
	}
}

// compile time check whether the TracingContext implements the TxContext interface
var _ TxContext = TracingContext{}
