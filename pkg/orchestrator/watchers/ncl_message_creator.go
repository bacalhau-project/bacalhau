package watchers

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol"
)

type NCLMessageCreatorFactory struct {
	protocolRouter *ProtocolRouter
	subjectFn      func(nodeID string) string
}

type NCLMessageCreatorFactoryParams struct {
	ProtocolRouter *ProtocolRouter
	SubjectFn      func(nodeID string) string
}

// NewNCLMessageCreatorFactory creates a new NCL protocol dispatcher factory
func NewNCLMessageCreatorFactory(params NCLMessageCreatorFactoryParams) *NCLMessageCreatorFactory {
	return &NCLMessageCreatorFactory{
		protocolRouter: params.ProtocolRouter,
		subjectFn:      params.SubjectFn,
	}
}

func (f *NCLMessageCreatorFactory) CreateMessageCreator(
	ctx context.Context, nodeID string) (nclprotocol.MessageCreator, error) {
	return NewNCLMessageCreator(NCLMessageCreatorParams{
		NodeID:         nodeID,
		ProtocolRouter: f.protocolRouter,
		SubjectFn:      f.subjectFn,
	})
}

type NCLMessageCreator struct {
	nodeID         string
	protocolRouter *ProtocolRouter
	subjectFn      func(nodeID string) string
}

type NCLMessageCreatorParams struct {
	NodeID         string
	ProtocolRouter *ProtocolRouter
	SubjectFn      func(nodeID string) string
}

// NewNCLMessageCreator creates a new NCL protocol dispatcher
func NewNCLMessageCreator(params NCLMessageCreatorParams) (*NCLMessageCreator, error) {
	err := errors.Join(
		validate.NotBlank(params.NodeID, "nodeID cannot be blank"),
		validate.NotNil(params.ProtocolRouter, "protocol router cannot be nil"),
		validate.NotNil(params.SubjectFn, "subject function cannot be nil"),
	)
	if params.SubjectFn != nil {
		// verify the subject function is provided and that it returns a non-empty string
		// by just validating against the current NodeID
		err = errors.Join(err,
			validate.NotBlank(params.SubjectFn(params.NodeID), "subject function returned empty"))
	}

	if err != nil {
		return nil, bacerrors.Wrap(err, "failed to create NCLMessageCreator").
			WithComponent(nclDispatcherErrComponent)
	}
	return &NCLMessageCreator{
		nodeID:         params.NodeID,
		protocolRouter: params.ProtocolRouter,
		subjectFn:      params.SubjectFn,
	}, nil
}

func (d *NCLMessageCreator) CreateMessage(event watcher.Event) (*envelope.Message, error) {
	upsert, ok := event.Object.(models.ExecutionUpsert)
	if !ok {
		return nil, bacerrors.Newf("failed to process event: expected models.ExecutionUpsert, got %T", event.Object).
			WithComponent(nclDispatcherErrComponent)
	}

	// Skip if there's no state change
	if !upsert.HasStateChange() {
		return nil, nil
	}
	if upsert.Current == nil {
		return nil, bacerrors.New("upsert.Current is nil").
			WithComponent(nclDispatcherErrComponent)
	}

	// Filter events not meant for the node this dispatcher is handling
	if upsert.Current.NodeID != d.nodeID {
		return nil, nil
	}

	execution := upsert.Current
	preferredProtocol, err := d.protocolRouter.PreferredProtocol(context.Background(), execution)
	if err != nil {
		return nil, bacerrors.Wrapf(err, "failed to determine preferred protocol for execution %s", execution.ID).
			WithComponent(bprotocolErrComponent)
	}
	if preferredProtocol != models.ProtocolNCLV1 {
		return nil, nil
	}

	transitions := newExecutionTransitions(upsert)
	var message *envelope.Message

	switch {
	case transitions.shouldAskForPendingBid():
		message = d.createAskForBidMessage(upsert)
	case transitions.shouldAskForDirectBid():
		message = d.createAskForBidMessage(upsert)
	case transitions.shouldAcceptBid():
		message = d.createBidAcceptedMessage(upsert)
	case transitions.shouldRejectBid():
		message = d.createBidRejectedMessage(upsert)
	case transitions.shouldCancel():
		message = d.createCancelMessage(upsert)
	}

	if message != nil {
		message.WithMetadataValue(ncl.KeySubject, d.subjectFn(upsert.Current.NodeID))
	}
	return message, nil
}

// transformNetworkConfig transforms the execution's network configurations for backward compatibility
// - If network type is host, change it to full
// - If network type is undefined, set the networkconfig to nil
// This method modifies the execution in place for efficiency since it's already a copy when used in createAskForBidMessage
func (d *NCLMessageCreator) transformNetworkConfig(execution *models.Execution) *models.Execution {
	// Only modify if we have a job
	if execution.Job == nil {
		return execution
	}

	// Get the first task (currently we only support one task)
	task := execution.Job.Task()
	if task == nil || task.Network == nil {
		// If task or network is already nil, nothing to do
		return execution
	}

	// Process the network configuration
	switch task.Network.Type {
	case models.NetworkHost:
		log.Trace().Msgf("Transforming network type from host to full for backward compatibility in execution %s", execution.ID)
		task.Network.Type = models.NetworkFull
	case models.NetworkDefault:
		log.Trace().Msgf("Setting undefined network type to nil for backward compatibility in execution %s", execution.ID)
		task.Network = nil
	}

	return execution
}

func (d *NCLMessageCreator) createAskForBidMessage(upsert models.ExecutionUpsert) *envelope.Message {
	log.Debug().
		Str("nodeID", upsert.Current.NodeID).
		Str("executionID", upsert.Current.ID).
		Msg("Asking for bid")

	// Apply network configuration transformation for backward compatibility
	transformedExecution := d.transformNetworkConfig(upsert.Current)

	return envelope.NewMessage(messages.AskForBidRequest{
		BaseRequest: messages.BaseRequest{Events: upsert.Events},
		Execution:   transformedExecution,
	}).WithMetadataValue(envelope.KeyMessageType, messages.AskForBidMessageType)
}

func (d *NCLMessageCreator) createBidAcceptedMessage(upsert models.ExecutionUpsert) *envelope.Message {
	log.Debug().
		Str("nodeID", upsert.Current.NodeID).
		Str("executionID", upsert.Current.ID).
		Msg("Accepting bid")

	return envelope.NewMessage(messages.BidAcceptedRequest{
		BaseRequest: messages.BaseRequest{Events: upsert.Events},
		ExecutionID: upsert.Current.ID,
		Accepted:    true,
	}).WithMetadataValue(envelope.KeyMessageType, messages.BidAcceptedMessageType)
}

func (d *NCLMessageCreator) createBidRejectedMessage(upsert models.ExecutionUpsert) *envelope.Message {
	log.Debug().
		Str("nodeID", upsert.Current.NodeID).
		Str("executionID", upsert.Current.ID).
		Msg("Rejecting bid")

	return envelope.NewMessage(messages.BidRejectedRequest{
		BaseRequest: messages.BaseRequest{Events: upsert.Events},
		ExecutionID: upsert.Current.ID,
	}).WithMetadataValue(envelope.KeyMessageType, messages.BidRejectedMessageType)
}

func (d *NCLMessageCreator) createCancelMessage(upsert models.ExecutionUpsert) *envelope.Message {
	log.Debug().
		Str("nodeID", upsert.Current.NodeID).
		Str("executionID", upsert.Current.ID).
		Msg("Cancelling execution")

	return envelope.NewMessage(messages.CancelExecutionRequest{
		BaseRequest: messages.BaseRequest{Events: upsert.Events},
		ExecutionID: upsert.Current.ID,
	}).WithMetadataValue(envelope.KeyMessageType, messages.CancelExecutionMessageType)
}

// compile-time check that NCLMessageCreator implements dispatcher.MessageCreator
var _ nclprotocol.MessageCreator = &NCLMessageCreator{}
var _ nclprotocol.MessageCreatorFactory = &NCLMessageCreatorFactory{}
