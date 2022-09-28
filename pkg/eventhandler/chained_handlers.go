package eventhandler

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

// An event handler implementation that chains multiple event handlers, and accepts a context provider
// to setup up the context once for all handlers.
// TODO: use generics when they are available instead of two separate types for local and job event handlers.
type ChainedLocalEventHandler struct {
	eventHandlers   []LocalEventHandler
	contextProvider system.ContextProvider
}

func NewChainedLocalEventHandler(contextProvider system.ContextProvider) *ChainedLocalEventHandler {
	return &ChainedLocalEventHandler{contextProvider: contextProvider}
}

func (r *ChainedLocalEventHandler) AddHandlers(handlers ...LocalEventHandler) {
	r.eventHandlers = append(r.eventHandlers, handlers...)
}

func (r *ChainedLocalEventHandler) HandleLocalEvent(ctx context.Context, event model.JobLocalEvent) error {
	if r.eventHandlers == nil {
		return fmt.Errorf("no event handlers registered")
	}

	jobCtx := r.contextProvider.GetContext(ctx, event.JobID)

	// All handlers are called, unless one of them returns an error.
	for _, handler := range r.eventHandlers {
		if err := handler.HandleLocalEvent(jobCtx, event); err != nil {
			return err
		}
	}
	return nil
}

// Job event handler chain
type ChainedJobEventHandler struct {
	eventHandlers   []JobEventHandler
	contextProvider system.ContextProvider
}

func NewChainedJobEventHandler(contextProvider system.ContextProvider) *ChainedJobEventHandler {
	return &ChainedJobEventHandler{contextProvider: contextProvider}
}

func (r *ChainedJobEventHandler) AddHandlers(handlers ...JobEventHandler) {
	r.eventHandlers = append(r.eventHandlers, handlers...)
}

func (r *ChainedJobEventHandler) HandleJobEvent(ctx context.Context, event model.JobEvent) error {
	if r.eventHandlers == nil {
		return fmt.Errorf("no event handlers registered")
	}

	jobCtx := r.contextProvider.GetContext(ctx, event.JobID)

	// All handlers are called, unless one of them returns an error.
	for _, handler := range r.eventHandlers {
		if err := handler.HandleJobEvent(jobCtx, event); err != nil {
			return err
		}
	}
	log.Trace().Msgf("handleJobEvent: %+v", event)
	return nil
}
