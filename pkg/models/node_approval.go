package models

import "fmt"

type NodeApproval struct {
	approval
}

type approval int

const (
	unknown approval = iota
	pending
	approved
	rejected
)

var (
	strApprovalArray = [...]string{
		pending:  "PENDING",
		approved: "APPROVED",
		rejected: "REJECTED",
	}

	typeApprovalMap = map[string]approval{
		"PENDING":  pending,
		"APPROVED": approved,
		"REJECTED": rejected,
	}
)

func (t approval) String() string {
	return strApprovalArray[t]
}

func Parse(a any) NodeApproval {
	switch v := a.(type) {
	case NodeApproval:
		return v
	case string:
		return NodeApproval{stringToApproval(v)}
	case fmt.Stringer:
		return NodeApproval{stringToApproval(v.String())}
	case int:
		return NodeApproval{approval(v)}
	case int64:
		return NodeApproval{approval(int(v))}
	case int32:
		return NodeApproval{approval(int(v))}
	}
	return NodeApproval{unknown}
}

func stringToApproval(s string) approval {
	if v, ok := typeApprovalMap[s]; ok {
		return v
	}
	return unknown
}

func (t approval) IsValid() bool {
	return t >= approval(1) && t <= approval(len(strApprovalArray))
}

type approvalsContainer struct {
	UNKNOWN  NodeApproval
	PENDING  NodeApproval
	APPROVED NodeApproval
	REJECTED NodeApproval
}

var NodeApprovals = approvalsContainer{
	UNKNOWN:  NodeApproval{unknown},
	PENDING:  NodeApproval{pending},
	APPROVED: NodeApproval{approved},
	REJECTED: NodeApproval{rejected},
}

func (c approvalsContainer) All() []NodeApproval {
	return []NodeApproval{
		c.PENDING,
		c.APPROVED,
		c.REJECTED,
	}
}

func (t NodeApproval) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.String() + `"`), nil
}

func (t *NodeApproval) UnmarshalJSON(b []byte) error {
	val := string(trimQuotes(b))
	*t = Parse(val)
	return nil
}

func trimQuotes(b []byte) []byte {
	if len(b) >= 2 {
		if b[0] == '"' && b[len(b)-1] == '"' {
			return b[1 : len(b)-1]
		}
	}
	return b
}
