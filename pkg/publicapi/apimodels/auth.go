package apimodels

import (
	"errors"

	"github.com/bacalhau-project/bacalhau/pkg/authn"
)

var ErrInvalidToken = errors.New("invalid token")

type ListAuthnMethodsRequest struct {
	BaseListRequest
}

type ListAuthnMethodsResponse struct {
	BaseListResponse
	Methods map[string]authn.Requirement
}

type AuthnRequest struct {
	BaseRequest
	Name       string
	MethodData []byte
}

type AuthnResponse struct {
	BaseResponse
	Authentication authn.Authentication
}
