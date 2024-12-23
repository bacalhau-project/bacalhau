package bprotocol

import (
	"errors"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages/legacy"
)

// CreateMessageRegistry creates a new payload registry.
func CreateMessageRegistry() (*envelope.Registry, error) {
	reg := envelope.NewRegistry()
	err := errors.Join(
		reg.Register(legacy.HeartbeatMessageType, legacy.Heartbeat{}),
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
