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
	if time.Since(t.startTime) > maxTracingDuration {
		log.Debug().Msgf("transaction took longer than %s to commit", maxTracingDuration.String())
	}
	return t.TxContext.Commit()
}

func (t TracingContext) Rollback() error {
	if time.Since(t.startTime) > maxTracingDuration {
		log.Debug().Msgf("transaction took longer than %s to rollback", maxTracingDuration.String())
	}
	return t.TxContext.Rollback()
}

// compile time check whether the TracingContext implements the TxContext interface
var _ TxContext = TracingContext{}
