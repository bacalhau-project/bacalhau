package eventhandler

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// An event handler implementation that chains multiple event handlers, and accepts a context provider
// to setup up the context once for all handlers.
type ChainedJobEventHandler struct {
	eventHandlers   []JobEventHandler
	contextProvider ContextProvider
}

func NewChainedJobEventHandler(contextProvider ContextProvider) *ChainedJobEventHandler {
	return &ChainedJobEventHandler{contextProvider: contextProvider}
}

func (r *ChainedJobEventHandler) AddHandlers(handlers ...JobEventHandler) {
	r.eventHandlers = append(r.eventHandlers, handlers...)
}

func (r *ChainedJobEventHandler) HandleJobEvent(ctx context.Context, event models.JobEvent) (err error) {
	startTime := time.Now()
	defer logEvent(ctx, event, startTime)(&err)

	if r.eventHandlers == nil {
		return fmt.Errorf("no event handlers registered")
	}

	jobCtx := r.contextProvider.GetContext(ctx, event.JobID)

	// All handlers are called, unless one of them returns an error.
	for _, handler := range r.eventHandlers {
		if err = handler.HandleJobEvent(jobCtx, event); err != nil { //nolint:gocritic
			return err
		}
	}
	return nil
}

func logEvent(ctx context.Context, event models.JobEvent, startTime time.Time) func(*error) {
	return func(handlerError *error) {
		logMsg := log.Ctx(ctx).Debug().
			Str("EventName", event.EventName.String()).
			Str("JobID", event.JobID).
			Str("NodeID", event.SourceNodeID).
			Str("Status", event.Status).
			Dur("HandleDuration", time.Since(startTime))
		if *handlerError != nil {
			logMsg = logMsg.AnErr("HandlerError", *handlerError)
		}

		logMsg.Msg("Handled event")
	}
}
