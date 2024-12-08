package bprotocol

import (
	"errors"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages/legacy"
)

// CreateMessageRegistry creates a new payload registry.
func CreateMessageRegistry() (*envelope.Registry, error) {
	reg := envelope.NewRegistry()
	err := errors.Join(
		reg.Register(legacy.HeartbeatMessageType, legacy.Heartbeat{}),
		reg.Register(messages.AskForBidMessageType, messages.AskForBidRequest{}),
		reg.Register(messages.BidAcceptedMessageType, messages.BidAcceptedRequest{}),
		reg.Register(messages.BidRejectedMessageType, messages.BidRejectedRequest{}),
		reg.Register(messages.CancelExecutionMessageType, messages.CancelExecutionRequest{}),
		reg.Register(messages.BidResultMessageType, messages.BidResult{}),
		reg.Register(messages.RunResultMessageType, messages.RunResult{}),
		reg.Register(messages.ComputeErrorMessageType, messages.ComputeError{}),
	)
	return reg, err
}

// MustCreateMessageRegistry creates a new payload registry.
func MustCreateMessageRegistry() *envelope.Registry {
	reg, err := CreateMessageRegistry()
	if err != nil {
		panic(err)
	}
	return reg
}
