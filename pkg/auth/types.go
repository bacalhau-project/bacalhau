package auth

import "net/http"

type AuthzDecision struct {
	Approved bool   `json:"approved"`
	Reason   string `json:"reason"`
}

type Authorizer interface {
	ShouldAllow(req *http.Request) (AuthzDecision, error)
}
