package requester

import (
	"fmt"

	"github.com/libp2p/go-libp2p/core/peer"
)

// ErrNotEnoughNodes is returned when not enough nodes in the network to run a job
type ErrNotEnoughNodes struct {
	RequestedNodes int
	AvailableNodes int
}

func NewErrNotEnoughNodes(requestedNodes, availableNodes int) ErrNotEnoughNodes {
	return ErrNotEnoughNodes{
		RequestedNodes: requestedNodes,
		AvailableNodes: availableNodes,
	}
}

func (e ErrNotEnoughNodes) Error() string {
	return fmt.Sprintf("not enough nodes to run job. requested: %d, available: %d", e.RequestedNodes, e.AvailableNodes)
}

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

type ErrJobAlreadyTerminal struct {
	JobID string
}

func NewErrJobAlreadyTerminal(jobID string) ErrJobAlreadyTerminal {
	return ErrJobAlreadyTerminal{JobID: jobID}
}

func (e ErrJobAlreadyTerminal) Error() string {
	return fmt.Errorf("job %s is already in a terminal state", e.JobID).Error()
}
