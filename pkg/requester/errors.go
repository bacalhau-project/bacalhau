package requester

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/libp2p/go-libp2p/core/peer"
)

// ErrNotEnoughNodes is returned when not enough nodes in the network to run a job
type ErrNotEnoughNodes struct {
	RequestedNodes int
	AvailableNodes []NodeRank
}

func NewErrNotEnoughNodes(requestedNodes int, availableNodes []NodeRank) ErrNotEnoughNodes {
	return ErrNotEnoughNodes{
		RequestedNodes: requestedNodes,
		AvailableNodes: availableNodes,
	}
}

func (e ErrNotEnoughNodes) Error() string {
	nodeErrors := ""
	available := 0
	for _, rank := range e.AvailableNodes {
		if rank.MeetsRequirement() {
			available += 1
		} else {
			nodeErrors += fmt.Sprintf("\n\tNode %s: %s", system.GetShortID(rank.NodeInfo.PeerInfo.ID.String()), rank.Reason)
		}
	}
	return fmt.Sprintf("not enough nodes to run job. requested: %d, available: %d. %s", e.RequestedNodes, available, nodeErrors)
}

// ErrTooManyRetries is returned when an execution has been retried too many times
type ErrTooManyRetries struct {
	Attempts int
}

func NewErrTooManyRetries(attempts int) ErrTooManyRetries {
	return ErrTooManyRetries{
		Attempts: attempts,
	}
}

func (e ErrTooManyRetries) Error() string {
	return fmt.Sprintf("too many retries (attempted %d)", e.Attempts)
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
