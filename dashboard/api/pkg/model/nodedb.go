package model

type ClusterMapNode struct {
	ID    string `json:"id"`
	Group int    `json:"group"`
}

type ClusterMapLink struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type ClusterMapResult struct {
	Nodes []ClusterMapNode `json:"nodes"`
	Links []ClusterMapLink `json:"links"`
}
