package eventhandler

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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

func (r *ChainedJobEventHandler) HandleJobEvent(ctx context.Context, event model.JobEvent) (err error) {
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

func logEvent(ctx context.Context, event model.JobEvent, startTime time.Time) func(*error) {
	return func(handlerError *error) {
		var logMsg *zerolog.Event

		// TODO: #829 Is checking environment every event the most efficient way
		// to do this? Could we just shunt logs to different places?
		switch system.GetEnvironment() {
		case system.EnvironmentDev, system.EnvironmentTest:
			logMsg = log.Ctx(ctx).Trace()
		default:
			logMsg = log.Ctx(ctx).Info()
		}

		logMsg = logMsg.
			Str("EventName", event.EventName.String()).
			Str("JobID", event.JobID).
			Int("ShardIndex", event.ShardIndex).
			Str("SourceNodeID", event.SourceNodeID).
			Str("TargetNodeID", event.TargetNodeID).
			Str("ClientID", event.ClientID).
			Str("Status", event.Status).
			Dur("HandleDuration", time.Since(startTime))
		if *handlerError != nil {
			logMsg = logMsg.AnErr("HandlerError", *handlerError)
		}

		logMsg.Msg("Handled event")
	}
}
