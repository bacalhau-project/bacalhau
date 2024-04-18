package models

import (
	"fmt"
)

// TODO if we ever pass a pointer to this type and use `==` comparison on it we're gonna have a bad time
// implement an `Equal()` method for this type and default to it.
type NodeMembershipState struct {
	membership
}

type membership int

const (
	unknown membership = iota
	pending
	approved
	rejected
)

var (
	strMembershipArray = [...]string{
		pending:  "PENDING",
		approved: "APPROVED",
		rejected: "REJECTED",
	}

	typeMembershipMap = map[string]membership{
		"PENDING":  pending,
		"APPROVED": approved,
		"REJECTED": rejected,
	}
)

func (t membership) String() string {
	return strMembershipArray[t]
}

func Parse(a any) NodeMembershipState {
	switch v := a.(type) {
	case NodeMembershipState:
		return v
	case string:
		return NodeMembershipState{stringToApproval(v)}
	case fmt.Stringer:
		return NodeMembershipState{stringToApproval(v.String())}
	case int:
		return NodeMembershipState{membership(v)}
	case int64:
		return NodeMembershipState{membership(int(v))}
	case int32:
		return NodeMembershipState{membership(int(v))}
	}
	return NodeMembershipState{unknown}
}

func stringToApproval(s string) membership {
	if v, ok := typeMembershipMap[s]; ok {
		return v
	}
	return unknown
}

func (t membership) IsValid() bool {
	return t >= membership(1) && t <= membership(len(strMembershipArray))
}

type membershipContainer struct {
	UNKNOWN  NodeMembershipState
	PENDING  NodeMembershipState
	APPROVED NodeMembershipState
	REJECTED NodeMembershipState
}

var NodeMembership = membershipContainer{
	UNKNOWN:  NodeMembershipState{unknown},
	PENDING:  NodeMembershipState{pending},
	APPROVED: NodeMembershipState{approved},
	REJECTED: NodeMembershipState{rejected},
}

func (c membershipContainer) All() []NodeMembershipState {
	return []NodeMembershipState{
		c.PENDING,
		c.APPROVED,
		c.REJECTED,
	}
}

func (t NodeMembershipState) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.String() + `"`), nil
}

func (t *NodeMembershipState) UnmarshalJSON(b []byte) error {
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
