package models

import (
	"fmt"
)

type NodeState struct {
	liveness
}

type liveness int

const (
	unknownState liveness = iota
	unhealthy
	healthy
)

var (
	strLivenessArray = [...]string{
		unknownState: "UNKNOWN",
		unhealthy:    "UNHEALTHY",
		healthy:      "HEALTHY",
	}

	typeLivenessMap = map[string]liveness{
		"UNKNOWN":   unknownState,
		"UNHEALTHY": unhealthy,
		"HEALTHY":   healthy,
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
	return NodeState{unknownState}
}

func stringToLiveness(s string) liveness {
	if v, ok := typeLivenessMap[s]; ok {
		return v
	}
	return unknownState
}

func (t liveness) IsValid() bool {
	return t >= liveness(1) && t <= liveness(len(strLivenessArray))
}

type livenessContainer struct {
	UNKNOWN   NodeState
	UNHEALTHY NodeState
	HEALTHY   NodeState
}

var NodeStates = livenessContainer{
	UNKNOWN:   NodeState{unknownState},
	UNHEALTHY: NodeState{unhealthy},
	HEALTHY:   NodeState{healthy},
}

func (c livenessContainer) All() []NodeState {
	return []NodeState{
		c.UNKNOWN,
		c.UNHEALTHY,
		c.HEALTHY,
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
