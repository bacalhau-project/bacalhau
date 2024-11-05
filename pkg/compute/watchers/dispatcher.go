package watchers

import (
	"context"
	"errors"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type Dispatcher struct {
	dispatchers map[models.Protocol]watcher.EventHandler
}

func NewDispatcher(dispatchers map[models.Protocol]watcher.EventHandler) (*Dispatcher, error) {
	err := errors.Join(
		validate.IsGreaterThanZero(len(dispatchers), "at least one protocol handler is required"),
	)
	for protocol, handler := range dispatchers {
		err = errors.Join(err, validate.NotNil(handler, "dispatcher for protocol %s cannot be nil", protocol))
	}
	if err != nil {
		return nil, bacerrors.Wrap(err, "failed to create Dispatcher").
			WithComponent(dispatcherErrComponent)
	}
	return &Dispatcher{
		dispatchers: dispatchers,
	}, nil
}

func (d *Dispatcher) HandleEvent(ctx context.Context, event watcher.Event) error {
	// Extract execution information from event
	upsert, ok := event.Object.(models.ExecutionUpsert)
	if !ok {
		return bacerrors.New("failed to process event: expected models.ExecutionUpsert, got %T", event.Object).
			WithComponent(dispatcherErrComponent)
	}

	protocol := d.determineProtocol(upsert.Current)
	dispatcher, ok := d.dispatchers[protocol]
	if !ok {
		return bacerrors.New("no dispatcher found for protocol %s", protocol).
			WithComponent(dispatcherErrComponent)
	}
	return dispatcher.HandleEvent(ctx, event)
}

func (d *Dispatcher) determineProtocol(execution *models.Execution) models.Protocol {
    if execution.Job == nil || execution.Job.Meta == nil {
        // Handle the nil Job or Meta appropriately
        // Return default protocol or an error
        return models.ProtocolBProtocolV2 // Default to legacy protocol
    }
    protocol, ok := execution.Job.Meta[models.MetaOrchestratorProtocol]
    if !ok {
        // TODO: Remove this once all jobs have the protocol set when v1.5 is no longer supported
        return models.ProtocolBProtocolV2 // Default to legacy protocol
    }
    return models.Protocol(protocol)
}
