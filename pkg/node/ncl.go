package node

import (
	"errors"

	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/node/heartbeat"
)

// CreatePayloadRegistry creates a new payload registry.
func CreatePayloadRegistry() (*ncl.PayloadRegistry, error) {
	reg := ncl.NewPayloadRegistry()
	err := errors.Join(
		reg.Register(heartbeat.HeartbeatMessageType, heartbeat.Heartbeat{}),
	)
	return reg, err
}
