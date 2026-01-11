package apimodels

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// IsAliveResponse is the response to the IsAlive request.
type IsAliveResponse struct {
	BaseGetResponse `json:",omitempty,inline" yaml:",omitempty,inline"`
	Status          string `json:"Status"`
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

type GetAgentConfigResponse struct {
	BaseGetResponse
	Config types.Bacalhau `json:"config"`
}

type GetAgentNodeAuthConfigResponse struct {
	BaseGetResponse
	Version string
	Config  types.Oauth2Config
}
