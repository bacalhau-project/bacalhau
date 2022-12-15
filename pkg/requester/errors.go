package requester

import "fmt"

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
