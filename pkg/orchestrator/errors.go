package orchestrator

import (
	"fmt"
	"strings"

	"github.com/samber/lo"

	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
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

func (e ErrNotEnoughNodes) SuitableNodes() int {
	return lo.CountBy(e.AvailableNodes, func(rank NodeRank) bool { return rank.MeetsRequirement() })
}

func (e ErrNotEnoughNodes) Error() string {
	reasons := lo.GroupBy(e.AvailableNodes, func(rank NodeRank) string { return rank.Reason })

	var message strings.Builder
	_, _ = fmt.Fprint(&message, "not enough nodes to run job. ")
	_, _ = fmt.Fprintf(&message, "requested: %d, available: %d, suitable: %d.", e.RequestedNodes, len(e.AvailableNodes), e.SuitableNodes())
	for reason, nodes := range reasons {
		_, _ = fmt.Fprint(&message, "\n• ")
		if len(nodes) > 1 {
			_, _ = fmt.Fprintf(&message, "%d of %d nodes", len(nodes), len(e.AvailableNodes))
		} else {
			_, _ = fmt.Fprintf(&message, "Node %s", idgen.ShortNodeID(nodes[0].NodeInfo.ID()))
		}
		_, _ = fmt.Fprintf(&message, ": %s", reason)
	}
	return message.String()
}

func (e ErrNotEnoughNodes) Retryable() bool {
	return lo.ContainsBy(e.AvailableNodes, func(rank NodeRank) bool {
		return !rank.MeetsRequirement() && rank.Retryable
	})
}

func (e ErrNotEnoughNodes) Details() map[string]string {
	return map[string]string{
		"NodesRequested": fmt.Sprint(e.RequestedNodes),
		"NodesAvailable": fmt.Sprint(len(e.AvailableNodes)),
		"NodesSuitable":  fmt.Sprint(e.SuitableNodes()),
	}
}

// ErrNoMatchingNodes is returned when no matching nodes in the network to run a job
type ErrNoMatchingNodes struct {
	AvailableNodes []NodeRank
}

func NewErrNoMatchingNodes(availableNodes []NodeRank) ErrNoMatchingNodes {
	return ErrNoMatchingNodes{
		AvailableNodes: availableNodes,
	}
}

func (e ErrNoMatchingNodes) Error() string {
	reasons := lo.GroupBy(e.AvailableNodes, func(rank NodeRank) string { return rank.Reason })

	var message strings.Builder
	_, _ = fmt.Fprintf(&message, "not matching nodes to run job out of %d available nodes.", len(e.AvailableNodes))
	for reason, nodes := range reasons {
		_, _ = fmt.Fprint(&message, "\n• ")
		if len(nodes) > 1 {
			_, _ = fmt.Fprintf(&message, "%d of %d nodes", len(nodes), len(e.AvailableNodes))
		} else {
			_, _ = fmt.Fprintf(&message, "Node %s", idgen.ShortNodeID(nodes[0].NodeInfo.ID()))
		}
		_, _ = fmt.Fprintf(&message, ": %s", reason)
	}
	return message.String()
}

func (e ErrNoMatchingNodes) Details() map[string]string {
	return map[string]string{
		"NodesAvailable": fmt.Sprint(len(e.AvailableNodes)),
	}
}

func (e ErrNoMatchingNodes) Retryable() bool {
	return lo.ContainsBy(e.AvailableNodes, func(rank NodeRank) bool {
		return !rank.MeetsRequirement() && rank.Retryable
	})
}
