package orchestrator

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/system"
)

// ErrSchedulerNotFound is returned when the scheduler is not found for a given evaluation type
type ErrSchedulerNotFound struct {
	EvaluationType string
}

func NewErrSchedulerNotFound(evaluationType string) ErrSchedulerNotFound {
	return ErrSchedulerNotFound{EvaluationType: evaluationType}
}

func (e ErrSchedulerNotFound) Error() string {
	return "scheduler not found for evaluation type: " + e.EvaluationType
}

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
