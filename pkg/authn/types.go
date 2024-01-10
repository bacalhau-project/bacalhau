package authn

import (
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
)

// The rule that authentication policies must implement. If the rule returns a
// token string, the authentication method has succeeded and passed policy, and
// the token string will be passed to future API calls. If the rule returns
// nothing, authentication has failed.
//
// This is typically provided by a package `bacalhau.authn` and defined rule
// `token`. See "challenge_ns_anon.rego" for a minimal example.
const PolicyTokenRule = "bacalhau.authn.token" //nolint:gosec

// Authentication represents the result of a user attempting to authenticate. If
// Success is true, Token will provide an access token that the user agent
// should pass to future API calls. If Success is false, Reason will provide a
// human-readable reason explaining why authentication failed.
type Authentication struct {
	Success bool   `json:"success"`
	Reason  string `json:"reason,omitempty"`
	Token   string `json:"token,omitempty"`
}

func Failed(reason string) Authentication {
	return Authentication{Success: false, Reason: reason}
}

func Error(err error) (Authentication, error) {
	return Failed(err.Error()), err
}

type MethodType string

const (
	// An authentication method that provides a challenge string that the user
	// must sign using their private key.
	MethodTypeChallenge MethodType = "challenge"
)

// Requirement represents information about how to authenticate using a
// configured method.
type Requirement struct {
	// The type of the method, informing the user agent how to prepare an
	// authentication response.
	Type MethodType `json:"type"`
	// Parameters specific to this authentication type. For example, a list of
	// required information, or minimum acceptable key sizes.
	Params any `json:"params"`
}

// Authenticator accepts HTTP requests for user authentications and returns the
// result of trying to authenticate the credentials supplied by the user.
type Authenticator interface {
	provider.Providable

	Authenticate(req *http.Request) (Authentication, error)
	Requirement() Requirement
}

// Provider maps "method names" to authenticator implementations. A method name
// is a human-readable string chosen by the person configuring the system that
// is shown to users to help them pick the authentication method they want to
// use. There can be multiple usages of the same Authenticator *type* but with
// different configs and parameters, each identified with a unique method name.
//
// For example, if an implementation wants to allow users to log in with Github
// or Bitbucket, they might both use an authenticator implementation of type
// "oidc", and each would appear once on this provider with key / method name
// "github" and "bitbucket".
type Provider = provider.Provider[Authenticator]
