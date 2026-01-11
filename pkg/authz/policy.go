package authz

import (
	"bytes"
	"crypto/rsa"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/lib/policy"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/samber/lo"
)

// The name of the rule that must be `true` for the authorization provider to
// permit access. This is typically provided by a policy with package name
// `bacalhau.authz` and then by defining a rule `allow`. See
// `policy_test_allow.rego` for a minimal example.
//
//nolint:gosec  // not hardcoded creds
const (
	AuthzAllowRule      = "bacalhau.authz.allow"
	AuthzTokenValidRule = "bacalhau.authz.token_valid"
)

type policyAuthorizer struct {
	policy *policy.Policy
	keyset string
	nodeID string

	allowQuery      policy.Query[authzData, bool]
	tokenValidQuery policy.Query[authzData, bool]
}

type httpData struct {
	Host    string      `json:"host"`
	Method  string      `json:"method"`
	Path    []string    `json:"path"`
	Query   url.Values  `json:"query"`
	Headers http.Header `json:"headers"`
	Body    string      `json:"body"`
}

type tokenData struct {
	Keyset   string `json:"cert"`
	Issuer   string `json:"iss"`
	Audience string `json:"aud"`
}

type authzData struct {
	HTTP        httpData  `json:"http"`
	Constraints tokenData `json:"constraints"`
}

//go:embed policies/*.rego
var policies embed.FS

// PolicyAuthorizer can authorize users by calling out to an external Rego
// policy containing logic to make decisions about who should be authorized.
func NewPolicyAuthorizer(authzPolicy *policy.Policy, key *rsa.PublicKey, nodeID string) Authorizer {
	p := &policyAuthorizer{
		policy:          authzPolicy,
		nodeID:          nodeID,
		allowQuery:      policy.AddQuery[authzData, bool](authzPolicy, AuthzAllowRule),
		tokenValidQuery: policy.AddQuery[authzData, bool](authzPolicy, AuthzTokenValidRule),
	}

	if key != nil {
		keys := jwk.NewSet()
		keys.Add(lo.Must(jwk.New(key)))

		var keyset strings.Builder
		lo.Must0(json.NewEncoder(&keyset).Encode(keys))
		p.keyset = keyset.String()
	}

	return p
}

// Authorize runs the loaded policy and provides a structure representing the
// inbound HTTP request as input.
func (authorizer *policyAuthorizer) Authorize(req *http.Request) (Authorization, error) {
	if req.URL == nil {
		return Authorization{}, errors.New("bad HTTP request: missing URL")
	}

	body := new(bytes.Buffer)
	if req.Body != nil {
		written, err := io.Copy(body, req.Body)
		if err != nil {
			return Authorization{}, err
		}
		defer func() { _ = req.Body.Close() }()
		if written != req.ContentLength {
			return Authorization{}, fmt.Errorf("read %d but was expecting %d", written, req.ContentLength)
		}

		// Put the Body back into a readable state.
		req.Body = io.NopCloser(body)
	}

	in := authzData{
		HTTP: httpData{
			Host:    req.Host,
			Method:  req.Method,
			Path:    strings.Split(strings.TrimLeft(req.URL.Path, "/"), "/"),
			Query:   req.URL.Query(),
			Headers: req.Header,
			Body:    body.String(),
		},
		// Metadata that can be used to verify the JWT, if it was signed by this
		// requester node (which does not have to be the case – users can submit
		// tokens signed elsewhere as long as the policy verifies them)
		Constraints: tokenData{
			Keyset:   authorizer.keyset,
			Issuer:   authorizer.nodeID,
			Audience: authorizer.nodeID,
		},
	}

	approved, aErr := authorizer.allowQuery(req.Context(), in)
	tokenValid, tvErr := authorizer.tokenValidQuery(req.Context(), in)
	return Authorization{Approved: approved, TokenValid: tokenValid}, errors.Join(aErr, tvErr)
}

// AlwaysAllowPolicy is a policy that will always permit access, irrespective of
// the passed in data, which is useful for testing.
var AlwaysAllowPolicy = lo.Must(policy.FromFS(policies, "policies/policy_test_allow.rego"))

// AlwaysAllow is an authorizer that will always permit access, irrespective of
// the passed in data, which is useful for testing.
var AlwaysAllow = NewPolicyAuthorizer(AlwaysAllowPolicy, nil, "")
