//go:build unit || !integration

package authz

import (
	"net/http"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/lib/policy"
	"github.com/stretchr/testify/require"
)

func TestAllowsWhenPolicySaysAllow(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "/api/v1/hello", nil)
	require.NoError(t, err)

	policy, err := policy.FromFS(policies, "policies/policy_test_allow.rego")
	require.NoError(t, err)

	authorizer := NewPolicyAuthorizer(policy)
	require.NotNil(t, authorizer)

	result, err := authorizer.Authorize(request)
	require.NoError(t, err)
	require.True(t, result.Approved)
}

func TestDeniesWhenPolicySaysDeny(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "/api/v1/hello", nil)
	require.NoError(t, err)

	policy, err := policy.FromFS(policies, "policies/policy_test_deny.rego")
	require.NoError(t, err)

	authorizer := NewPolicyAuthorizer(policy)
	require.NotNil(t, authorizer)

	result, err := authorizer.Authorize(request)
	require.NoError(t, err)
	require.False(t, result.Approved)
}

func TestPolicyEvaluatedAgainstHTTPRequest(t *testing.T) {
	policy, err := policy.FromFS(policies, "policies/policy_test_http.rego")
	require.NoError(t, err)

	authorizer := NewPolicyAuthorizer(policy)
	require.NotNil(t, authorizer)

	goodRequest, err := http.NewRequest(http.MethodGet, "/api/v1/hello", nil)
	require.NoError(t, err)

	goodResult, err := authorizer.Authorize(goodRequest)
	require.NoError(t, err)
	require.True(t, goodResult.Approved)

	badRequest, err := http.NewRequest(http.MethodDelete, "/api/v1/hello", nil)
	require.NoError(t, err)

	badResult, err := authorizer.Authorize(badRequest)
	require.NoError(t, err)
	require.False(t, badResult.Approved)
}
