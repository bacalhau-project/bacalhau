package eventhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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
		// construct log event
		logData := eventLog{
			EventName:    event.EventName,
			ShardID:      fmt.Sprintf("%s_%d", event.JobID, event.ShardIndex),
			SourceNodeID: event.SourceNodeID,
			TargetNodeID: event.TargetNodeID,
			Duration:     time.Since(startTime).Milliseconds(),
			ClientID:     event.ClientID,
			Status:       event.Status,
		}

		if *handlerError != nil {
			logData.HandlerError = (*handlerError).Error()
		}

		jsonBytes, err := json.Marshal(logData)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("failed to marshal event for logging purposes: %+v", event)
		}

		// log event
		if *handlerError != nil {
			log.Ctx(ctx).Error().Msg(string(jsonBytes))
		} else {
			log.Ctx(ctx).Info().Msg(string(jsonBytes))
		}
	}
}

type eventLog struct {
	EventName    model.JobEventType `json:"EventName"`
	ShardID      string             `json:"ShardID"`
	SourceNodeID string             `json:"SourceNodeID"`
	TargetNodeID string             `json:"TargetNodeID"`
	ClientID     string             `json:"ClientID,omitempty"`
	Status       string             `json:"Status,omitempty"`
	Duration     int64              `json:"Duration"`
	HandlerError string             `json:"HandlerError,omitempty"`
}
