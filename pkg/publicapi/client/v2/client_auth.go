package client

import "github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"

const authBase = "/api/v1/auth"

type Auth struct {
	client *Client
}

func (c *Client) Auth() *Auth {
	return &Auth{client: c}
}

func (auth *Auth) Methods(r *apimodels.ListAuthnMethodsRequest) (*apimodels.ListAuthnMethodsResponse, error) {
	var resp apimodels.ListAuthnMethodsResponse
	err := auth.client.list(authBase, r, &resp)
	return &resp, err
}

func (auth *Auth) Authenticate(r *apimodels.AuthnRequest) (*apimodels.AuthnResponse, error) {
	var resp apimodels.AuthnResponse
	err := auth.client.post(authBase+"/"+r.Name, r, &resp)
	return &resp, err
}
