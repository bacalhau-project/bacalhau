package orchestrator

import (
	"fmt"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
	"github.com/samber/lo"
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
	suitable := lo.CountBy(e.AvailableNodes, func(rank NodeRank) bool { return rank.MeetsRequirement() })
	reasons := lo.GroupBy(e.AvailableNodes, func(rank NodeRank) string { return rank.Reason })

	var message strings.Builder
	fmt.Fprint(&message, "not enough nodes to run job. ")
	fmt.Fprintf(&message, "requested: %d, available: %d, suitable: %d.", e.RequestedNodes, len(e.AvailableNodes), suitable)
	for reason, nodes := range reasons {
		fmt.Fprint(&message, "\n• ")
		if len(nodes) > 1 {
			fmt.Fprintf(&message, "%d of %d nodes", len(nodes), len(e.AvailableNodes))
		} else {
			fmt.Fprintf(&message, "Node %s", idgen.ShortNodeID(nodes[0].NodeInfo.ID()))
		}
		fmt.Fprintf(&message, ": %s", reason)
	}
	return message.String()
}

// ErrNoMatchingNodes is returned when no matching nodes in the network to run a job
type ErrNoMatchingNodes struct {
}

func NewErrNoMatchingNodes() ErrNoMatchingNodes {
	return ErrNoMatchingNodes{}
}

func (e ErrNoMatchingNodes) Error() string {
	return "no matching nodes to run job"
}
