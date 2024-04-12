package models

// NodeState contains metadata about the state of a node on the network. Requester nodes maintain a NodeState for
// each node they are aware of. The NodeState represents a Requester nodes view of another node on the network.
type NodeState struct {
	Info     NodeInfo     `json:"Info"`
	Approval NodeApproval `json:"Approval"`
	Liveness NodeLiveness `json:"Liveness"`
}
