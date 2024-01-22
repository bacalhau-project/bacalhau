package authz

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/lib/policy"
	"github.com/samber/lo"
)

// The name of the rule that must be `true` for the authorization provider to
// permit access. This is typically provided by a policy with package name
// `bacalhau.authz` and then by defining a rule `allow`. See
// `policy_test_allow.rego` for a minimal example.
const AuthzAllowRule = "bacalhau.authz.allow"

type policyAuthorizer struct {
	policy     *policy.Policy
	allowQuery policy.Query[authzData, bool]
}

type httpData struct {
	Host    string      `json:"host"`
	Method  string      `json:"method"`
	Path    []string    `json:"path"`
	Query   url.Values  `json:"query"`
	Headers http.Header `json:"headers"`
	Body    string      `json:"body"`
}

type authzData struct {
	HTTP httpData `json:"http"`
}

//go:embed policies/*.rego
var policies embed.FS

// PolicyAuthorizer can authorize users by calling out to an external Rego
// policy containing logic to make decisions about who should be authorized.
func NewPolicyAuthorizer(authzPolicy *policy.Policy) Authorizer {
	return &policyAuthorizer{
		policy:     authzPolicy,
		allowQuery: policy.AddQuery[authzData, bool](authzPolicy, AuthzAllowRule),
	}
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
		defer req.Body.Close()
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
	}

	approved, err := authorizer.allowQuery(req.Context(), in)
	return Authorization{Approved: approved}, err
}

// AlwaysAllow is an authorizer that will always permit access, irrespective of
// the passed in data, which is useful for testing.
var AlwaysAllow = NewPolicyAuthorizer(lo.Must(policy.FromFS(policies, "policies/policy_test_allow.rego")))
