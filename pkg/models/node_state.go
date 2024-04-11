package models

type NodeState struct {
	Info     NodeInfo     `json:"Info"`
	Approval NodeApproval `json:"Approval"`
	Liveness NodeLiveness `json:"Liveness"`
}
