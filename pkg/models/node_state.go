package models

import (
	"fmt"
)

type NodeState struct {
	liveness
}

type liveness int

// To add a new state (for instance, a state beyond which the node is considered
// lost) then:
// * add it to the end of the list in the const below
// * add it to strLivenessArray and typeLivenessMap
// * add it to the livenessContainer and corresponding NodeStates var.
// * add it to the All() method in the livenessContainer
const (
	connected liveness = iota
	disconnected
)

var (
	strLivenessArray = [...]string{
		connected:    "CONNECTED",
		disconnected: "DISCONNECTED",
	}

	typeLivenessMap = map[string]liveness{
		"CONNECTED":    connected,
		"DISCONNECTED": disconnected,
	}
)

func (t liveness) String() string {
	return strLivenessArray[t]
}

func ParseState(a any) NodeState {
	switch v := a.(type) {
	case NodeState:
		return v
	case string:
		return NodeState{stringToLiveness(v)}
	case fmt.Stringer:
		return NodeState{stringToLiveness(v.String())}
	case int:
		return NodeState{liveness(v)}
	case int64:
		return NodeState{liveness(int(v))}
	case int32:
		return NodeState{liveness(int(v))}
	}
	return NodeState{disconnected}
}

func stringToLiveness(s string) liveness {
	if v, ok := typeLivenessMap[s]; ok {
		return v
	}
	return disconnected
}

func (t liveness) IsValid() bool {
	return t >= liveness(1) && t <= liveness(len(strLivenessArray))
}

type livenessContainer struct {
	CONNECTED    NodeState
	DISCONNECTED NodeState
	HEALTHY      NodeState
}

var NodeStates = livenessContainer{
	CONNECTED:    NodeState{connected},
	DISCONNECTED: NodeState{disconnected},
}

func (c livenessContainer) All() []NodeState {
	return []NodeState{
		c.CONNECTED,
		c.DISCONNECTED,
	}
}

func (s NodeState) MarshalJSON() ([]byte, error) {
	return []byte(`"` + s.String() + `"`), nil
}

func (s *NodeState) UnmarshalJSON(b []byte) error {
	val := string(trimQuotes(b))
	*s = ParseState(val)
	return nil
}
