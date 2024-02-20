package authz

import (
	"net/http"
)

type Authorization struct {
	Approved   bool   `json:"approved"`
	TokenValid bool   `json:"tokenValid"`
	Reason     string `json:"reason"`
}

type Authorizer interface {
	Authorize(req *http.Request) (Authorization, error)
}
