package client

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
)

const authBase = "/api/v1/auth"

type Auth struct {
	client *Client
}

func (c *Client) Auth() *Auth {
	return &Auth{client: c}
}

func (auth *Auth) Methods(ctx context.Context, r *apimodels.ListAuthnMethodsRequest) (*apimodels.
	ListAuthnMethodsResponse, error) {
	var resp apimodels.ListAuthnMethodsResponse
	err := auth.client.list(ctx, authBase, r, &resp)
	return &resp, err
}

func (auth *Auth) Authenticate(ctx context.Context, r *apimodels.AuthnRequest) (*apimodels.AuthnResponse, error) {
	var resp apimodels.AuthnResponse
	err := auth.client.post(ctx, authBase+"/"+r.Name, r, &resp)
	return &resp, err
}
