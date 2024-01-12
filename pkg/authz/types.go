package authz

import (
	"net/http"
)

type Authorization struct {
	Approved bool   `json:"approved"`
	Reason   string `json:"reason"`
}

type Authorizer interface {
	Authorize(req *http.Request) (Authorization, error)
}
