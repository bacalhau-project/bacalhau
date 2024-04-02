package model

import "context"

type DebugInfoProvider interface {
	GetDebugInfo(ctx context.Context) (DebugInfo, error)
}

type DebugInfo struct {
	Component string      `json:"component"`
	Info      interface{} `json:"info"`
}
