package client

import (
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
)

type Agent struct {
	client *Client
}

// Agent returns a handle on the agent endpoints.
func (c *Client) Agent() *Agent {
	return &Agent{client: c}
}

// Alive is used to check if the agent is alive.
func (c *Agent) Alive() (*apimodels.IsAliveResponse, error) {
	req := &apimodels.BaseGetRequest{
		BaseRequest: apimodels.BaseRequest{
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
	}
	var res apimodels.IsAliveResponse
	err := c.client.get("/api/v1/agent/alive", req, &res)
	return &res, err
}

// Version is used to get the agent version.
func (c *Agent) Version() (*apimodels.GetVersionResponse, error) {
	req := &apimodels.BaseGetRequest{
		BaseRequest: apimodels.BaseRequest{
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
	}
	var res apimodels.GetVersionResponse
	err := c.client.get("/api/v1/agent/version", req, &res)
	return &res, err
}

// Node is used to get the agent node info.
func (c *Agent) Node(req *apimodels.GetAgentNodeRequest) (*apimodels.GetAgentNodeResponse, error) {
	var res apimodels.GetAgentNodeResponse
	err := c.client.get("/api/v1/agent/node", req, &res)
	return &res, err
}
