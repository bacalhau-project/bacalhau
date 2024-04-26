package models

import (
	"fmt"
)

// TODO if we ever pass a pointer to this type and use `==` comparison on it we're gonna have a bad time
// implement an `Equal()` method for this type and default to it.
type NodeConnectionState struct {
	connection
}

type connection int

// To add a new state (for instance, a state beyond which the node is considered
// lost) then:
// * add it to the end of the list in the const below
// * add it to strConnectionArray and typeConnectionMap
// * add it to the livenessContainer and corresponding NodeStates var.
// * add it to the All() method in the livenessContainer
const (
	connected connection = iota
	disconnected
)

var (
	strConnectionArray = [...]string{
		connected:    "CONNECTED",
		disconnected: "DISCONNECTED",
	}

	typeConnectionMap = map[string]connection{
		"CONNECTED":    connected,
		"DISCONNECTED": disconnected,
	}
)

func (t connection) String() string {
	return strConnectionArray[t]
}

func ParseConnection(a any) NodeConnectionState {
	switch v := a.(type) {
	case NodeConnectionState:
		return v
	case string:
		return NodeConnectionState{stringToConnection(v)}
	case fmt.Stringer:
		return NodeConnectionState{stringToConnection(v.String())}
	case int:
		return NodeConnectionState{connection(v)}
	case int64:
		return NodeConnectionState{connection(int(v))}
	case int32:
		return NodeConnectionState{connection(int(v))}
	}
	return NodeConnectionState{disconnected}
}

func stringToConnection(s string) connection {
	if v, ok := typeConnectionMap[s]; ok {
		return v
	}
	return disconnected
}

func (t connection) IsValid() bool {
	return t >= connection(1) && t <= connection(len(strConnectionArray))
}

type livenessContainer struct {
	CONNECTED    NodeConnectionState
	DISCONNECTED NodeConnectionState
	HEALTHY      NodeConnectionState
}

var NodeStates = livenessContainer{
	CONNECTED:    NodeConnectionState{connected},
	DISCONNECTED: NodeConnectionState{disconnected},
}

func (c livenessContainer) All() []NodeConnectionState {
	return []NodeConnectionState{
		c.CONNECTED,
		c.DISCONNECTED,
	}
}

func (s NodeConnectionState) MarshalJSON() ([]byte, error) {
	return []byte(`"` + s.String() + `"`), nil
}

func (s *NodeConnectionState) UnmarshalJSON(b []byte) error {
	val := string(trimQuotes(b))
	*s = ParseConnection(val)
	return nil
}
