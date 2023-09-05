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
	Labels []*labels.Requirement `query:"-"` // don't auto bind as it requires special handling
}

// ToHTTPRequest is used to convert the request to an HTTP request
func (o *ListNodesRequest) ToHTTPRequest() *HTTPRequest {
	r := o.BaseListRequest.ToHTTPRequest()

	for _, v := range o.Labels {
		r.Params.Add("labels", v.String())
	}
	return r
}

type ListNodesResponse struct {
	BaseListResponse
	Nodes []*models.NodeInfo
}
