package model

type DebugInfoProvider interface {
	GetDebugInfo() (DebugInfo, error)
}

type DebugInfo struct {
	Component string `json:"component"`
	Info      string `json:"info"`
}
