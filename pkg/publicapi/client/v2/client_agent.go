package client

import (
	"context"

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
func (c *Agent) Alive(ctx context.Context) (*apimodels.IsAliveResponse, error) {
	var res apimodels.IsAliveResponse
	err := c.client.get(ctx, "/api/v1/agent/alive", &apimodels.BaseGetRequest{}, &res)
	return &res, err
}

// Version is used to get the agent version.
func (c *Agent) Version(ctx context.Context) (*apimodels.GetVersionResponse, error) {
	var res apimodels.GetVersionResponse
	err := c.client.get(ctx, "/api/v1/agent/version", &apimodels.BaseGetRequest{}, &res)
	return &res, err
}

// Node is used to get the agent node info.
func (c *Agent) Node(ctx context.Context, req *apimodels.GetAgentNodeRequest) (*apimodels.GetAgentNodeResponse, error) {
	var res apimodels.GetAgentNodeResponse
	err := c.client.get(ctx, "/api/v1/agent/node", req, &res)
	return &res, err
}
