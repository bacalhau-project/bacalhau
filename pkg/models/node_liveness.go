package models

import (
	"fmt"
)

// TODO if we ever pass a pointer to this type and use `==` comparison on it we're gonna have a bad time
// implement an `Equal()` method for this type and default to it.
type NodeLiveness struct {
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

func ParseState(a any) NodeLiveness {
	switch v := a.(type) {
	case NodeLiveness:
		return v
	case string:
		return NodeLiveness{stringToLiveness(v)}
	case fmt.Stringer:
		return NodeLiveness{stringToLiveness(v.String())}
	case int:
		return NodeLiveness{liveness(v)}
	case int64:
		return NodeLiveness{liveness(int(v))}
	case int32:
		return NodeLiveness{liveness(int(v))}
	}
	return NodeLiveness{disconnected}
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
	CONNECTED    NodeLiveness
	DISCONNECTED NodeLiveness
	HEALTHY      NodeLiveness
}

var NodeStates = livenessContainer{
	CONNECTED:    NodeLiveness{connected},
	DISCONNECTED: NodeLiveness{disconnected},
}

func (c livenessContainer) All() []NodeLiveness {
	return []NodeLiveness{
		c.CONNECTED,
		c.DISCONNECTED,
	}
}

func (s NodeLiveness) MarshalJSON() ([]byte, error) {
	return []byte(`"` + s.String() + `"`), nil
}

func (s *NodeLiveness) UnmarshalJSON(b []byte) error {
	val := string(trimQuotes(b))
	*s = ParseState(val)
	return nil
}
