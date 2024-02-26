package models

import "fmt"

// NodeApproval is used to denote the approval status of a given
// node. These values are set based on the approval process for
// nodes which will auto-approve some, rely on human approval for
// others as well as rejecting some nodes as inelligible.
type NodeApproval string

const (
	NodeApprovalUnknown  NodeApproval = "unknown"
	NodeApprovalPending  NodeApproval = "pending"
	NodeApprovalApproved NodeApproval = "approved"
	NodeApprovalRejected NodeApproval = "rejected"
)

// ParseApproval accepts a string representation of a NodeApproval
// and returns the matching NodeApproval value, or an error if
// the string cannot be matched to a NodeApproval value.
func ParseApproval(s string) (NodeApproval, error) {
	switch s {
	case "pending":
		return NodeApprovalPending, nil
	case "approved":
		return NodeApprovalApproved, nil
	case "rejected":
		return NodeApprovalRejected, nil
	case "unknown":
		return NodeApprovalUnknown, nil
	default:
		return NodeApprovalUnknown, fmt.Errorf("invalid approval status: %s", s)
	}
}

// String returns the string representation of the NodeApproval
func (e NodeApproval) String() string {
	return string(e)
}
