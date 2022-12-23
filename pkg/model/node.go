package model

import (
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

type NodeInfoProvider interface {
	GetNodeInfo() (NodeInfo, error)
}

type NodeInfo struct {
	Component string `json:"component"`
	Info      string `json:"info"`
}

type NodeEvent struct {
	EventTime         time.Time            `json:"EventTime,omitempty"`
	NodeID            string               `json:"NodeID,omitempty" example:"9304c616-291f-41ad-b862-54e133c0149e"`
	EventName         NodeEventType        `json:"EventName,omitempty"`
	TotalCapacity     ResourceUsageData    `json:"TotalCapacity,omitempty"`
	AvailableCapacity ResourceUsageData    `json:"AvailableCapacity,omitempty"`
	Peers             map[string][]peer.ID `json:"Peers,omitempty"`
	DebugInfo         []DebugInfo          `json:"DebugInfo,omitempty"`
}
