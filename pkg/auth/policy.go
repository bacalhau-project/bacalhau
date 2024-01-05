package auth

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"github.com/open-policy-agent/opa/loader"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/topdown"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"golang.org/x/exp/maps"
)

// The name of the rule that must be `true` for the authorization provider to
// permit access. This is typically provided by a policy with package name
// `bacalhau.authz` and then by defining a rule `allow`. See
// `policy_test_allow.rego` for a minimal example.
const AuthzAllowRule = "bacalhau.authz.allow"

type policyAuthorizer struct {
	allowQuery rego.PreparedEvalQuery
}

type regoOpt = func(*rego.Rego)

//go:embed policies/*.rego
var policies embed.FS

// PolicyAuthorizer can authorize users by calling out to an external Rego
// policy containing logic to make decisions about who should be authorized.
func NewPolicyAuthorizer(policySource fs.FS, policyPath string) (Authorizer, error) {
	results, err := loader.NewFileLoader().WithFS(policySource).All([]string{policyPath})
	if err != nil {
		return nil, err
	}

	opts := []regoOpt{
		rego.Query("data." + AuthzAllowRule),
	}
	modules := lo.Map(maps.Values(results.Modules), func(m *loader.RegoFile, _ int) regoOpt { return rego.ParsedModule(m.Parsed) })
	query := rego.New(append(opts, modules...)...)

	allowQuery, err := query.PrepareForEval(context.TODO())
	return &policyAuthorizer{allowQuery: allowQuery}, err
}

// ShouldAllow runs the loaded policy and provides a structure representing the
// inbound HTTP request as input.
func (authorizer *policyAuthorizer) ShouldAllow(req *http.Request) (AuthzDecision, error) {
	if req.URL == nil {
		return AuthzDecision{}, errors.New("bad HTTP request: missing URL")
	}

	body := new(bytes.Buffer)
	if req.Body != nil {
		written, err := io.Copy(body, req.Body)
		if err != nil {
			return AuthzDecision{}, err
		} else if written != req.ContentLength {
			return AuthzDecision{}, fmt.Errorf("read %d but was expecting %d", written, req.ContentLength)
		}
		defer req.Body.Close()

		// Put the Body back into a readable state.
		req.Body = io.NopCloser(body)
	}

	in := map[string]interface{}{
		"http": map[string]interface{}{
			"host":    req.Host,
			"method":  req.Method,
			"path":    strings.Split(strings.TrimLeft(req.URL.Path, "/"), "/"),
			"query":   req.URL.Query(),
			"headers": req.Header,
			"body":    body.String(),
		},
	}

	tracer := topdown.NewBufferTracer()
	results, err := authorizer.allowQuery.Eval(req.Context(), rego.EvalInput(in), rego.EvalQueryTracer(tracer))
	if err != nil {
		return AuthzDecision{}, err
	}

	// Output tracing information, but only if the log level is appropriate
	// So we avoid going into a long loop of no-ops
	if logger := log.Ctx(req.Context()); logger.GetLevel() <= zerolog.TraceLevel {
		for _, event := range *tracer {
			logger.Trace().Str("Op", string(event.Op)).Str("Eval", event.Node.String()).Send()
		}
	}

	return AuthzDecision{
		Approved: results.Allowed(),
	}, nil
}

// AlwaysAllow is an authorizer that will always permit access, irrespective of
// the passed in data, which is useful for testing.
var AlwaysAllow = lo.Must(NewPolicyAuthorizer(policies, "policies/policy_test_allow.rego"))
