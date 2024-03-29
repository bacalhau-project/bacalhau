package apimodels

import (
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"k8s.io/apimachinery/pkg/labels"
)

type GetNodeRequest struct {
	BaseGetRequest
	NodeID string
}

type GetNodeResponse struct {
	BaseGetResponse
	Node *models.NodeInfo
}

type ListNodesRequest struct {
	BaseListRequest
	Labels         []labels.Requirement `query:"-"` // don't auto bind as it requires special handling
	FilterByStatus string               `query:"filter-status"`
}

// ToHTTPRequest is used to convert the request to an HTTP request
func (o *ListNodesRequest) ToHTTPRequest() *HTTPRequest {
	r := o.BaseListRequest.ToHTTPRequest()

	for _, v := range o.Labels {
		r.Params.Add("labels", v.String())
	}

	if o.FilterByStatus != "" {
		r.Params.Add("filter-status", o.FilterByStatus)
	}

	return r
}

type ListNodesResponse struct {
	BaseListResponse
	Nodes []*models.NodeInfo
}

type PutNodeRequest struct {
	BasePutRequest
	Action  string
	Message string
	NodeID  string
}

type PutNodeResponse struct {
	BasePutResponse
	Success bool
	Error   string
}

type NodeAction string

const (
	NodeActionApprove NodeAction = "approve"
	NodeActionReject  NodeAction = "reject"
)

func (n NodeAction) Description() string {
	switch n {
	case NodeActionApprove:
		return "Approve a node whose membership is pending"
	case NodeActionReject:
		return "Reject a node whose membership is pending"
	}
	return ""
}

func (n NodeAction) IsValid() bool {
	return n == NodeActionApprove || n == NodeActionReject
}
