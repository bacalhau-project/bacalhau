package routing

import (
	"fmt"

	"github.com/libp2p/go-libp2p/core/peer"
)

// ErrNodeNotFound is returned when nodeInfo was not found for a requested peer id
type ErrNodeNotFound struct {
	peerID peer.ID
}

func NewErrNodeNotFound(peerID peer.ID) ErrNodeNotFound {
	return ErrNodeNotFound{peerID: peerID}
}

func (e ErrNodeNotFound) Error() string {
	return fmt.Errorf("nodeInfo not found for peer id: %s", e.peerID).Error()
}
