package node

import (
	"errors"

	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/node/heartbeat"
)

// CreateMessageSerDeRegistry creates a new payload registry.
func CreateMessageSerDeRegistry() (*ncl.MessageSerDeRegistry, error) {
	reg := ncl.NewMessageSerDeRegistry()
	err := errors.Join(
		reg.Register(heartbeat.HeartbeatMessageType, heartbeat.Heartbeat{}),
	)
	return reg, err
}
