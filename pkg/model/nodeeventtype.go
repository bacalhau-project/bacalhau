package model

import "fmt"

//go:generate stringer -type=NodeEventType --trimprefix=NodeEvent
type NodeEventType int

const (
	nodeEventUnknown NodeEventType = iota // must be first

	NodeEventAnnounce

	nodeEventDone // must be last
)

func ParseNodeEventType(str string) (NodeEventType, error) {
	for typ := nodeEventUnknown + 1; typ < nodeEventDone; typ++ {
		if equal(typ.String(), str) {
			return typ, nil
		}
	}

	return nodeEventUnknown, fmt.Errorf(
		"executor: unknown job event type '%s'", str)
}

func NodeEventTypes() []NodeEventType {
	var res []NodeEventType
	for typ := nodeEventUnknown + 1; typ < nodeEventDone; typ++ {
		res = append(res, typ)
	}

	return res
}

func (je NodeEventType) MarshalText() ([]byte, error) {
	return []byte(je.String()), nil
}

func (je *NodeEventType) UnmarshalText(text []byte) (err error) {
	name := string(text)
	*je, err = ParseNodeEventType(name)
	return
}
