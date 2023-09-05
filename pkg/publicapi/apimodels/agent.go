package apimodels

import "github.com/bacalhau-project/bacalhau/pkg/models"

// IsAliveResponse is the response to the IsAlive request.
type IsAliveResponse struct {
	BaseGetResponse
	Status string
}

func (r *IsAliveResponse) IsReady() bool {
	if r != nil && r.Status == "OK" {
		return true
	}
	return false
}

// GetVersionResponse is the response to the Version request.
type GetVersionResponse struct {
	BaseGetResponse
	*models.BuildVersionInfo
}

// GetAgentNodeRequest is the request to get the agent node.
type GetAgentNodeRequest struct {
	BaseGetRequest
}

type GetAgentNodeResponse struct {
	BaseGetResponse
	*models.NodeInfo
}
