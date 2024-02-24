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

	authorizer := NewPolicyAuthorizer(policy, nil, "")
	require.NotNil(t, authorizer)

	result, err := authorizer.Authorize(request)
	require.NoError(t, err)
	require.True(t, result.Approved)
	require.True(t, result.TokenValid)
}

func TestDeniesWhenPolicySaysDeny(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "/api/v1/hello", nil)
	require.NoError(t, err)

	policy, err := policy.FromFS(policies, "policies/policy_test_deny.rego")
	require.NoError(t, err)

	authorizer := NewPolicyAuthorizer(policy, nil, "")
	require.NotNil(t, authorizer)

	result, err := authorizer.Authorize(request)
	require.NoError(t, err)
	require.False(t, result.Approved)
	require.True(t, result.TokenValid)
}

func TestPolicyEvaluatedAgainstGoodHTTPRequest(t *testing.T) {
	policy, err := policy.FromFS(policies, "policies/policy_test_http.rego")
	require.NoError(t, err)

	authorizer := NewPolicyAuthorizer(policy, nil, "")
	require.NotNil(t, authorizer)

	goodRequest, err := http.NewRequest(http.MethodGet, "/api/v1/hello", nil)
	require.NoError(t, err)
	goodRequest.Header.Add("Authorization", "anything")

	goodResult, err := authorizer.Authorize(goodRequest)
	require.NoError(t, err)
	require.True(t, goodResult.Approved)
	require.True(t, goodResult.TokenValid)
}

func TestPolicyEvaluatedAgainstInvalidHTTPRequest(t *testing.T) {
	policy, err := policy.FromFS(policies, "policies/policy_test_http.rego")
	require.NoError(t, err)

	authorizer := NewPolicyAuthorizer(policy, nil, "")
	require.NotNil(t, authorizer)

	invalidRequest, err := http.NewRequest(http.MethodGet, "/api/v1/hello", nil)
	require.NoError(t, err)
	// Not adding authz header

	invalidResult, err := authorizer.Authorize(invalidRequest)
	require.NoError(t, err)
	require.False(t, invalidResult.Approved)
	require.False(t, invalidResult.TokenValid)
}

func TestPolicyEvaluatedAgainstDeniedHTTPRequest(t *testing.T) {
	policy, err := policy.FromFS(policies, "policies/policy_test_http.rego")
	require.NoError(t, err)

	authorizer := NewPolicyAuthorizer(policy, nil, "")
	require.NotNil(t, authorizer)

	badRequest, err := http.NewRequest(http.MethodDelete, "/api/v1/hello", nil)
	require.NoError(t, err)
	badRequest.Header.Add("Authorization", "anything")

	badResult, err := authorizer.Authorize(badRequest)
	require.NoError(t, err)
	require.False(t, badResult.Approved)
	require.True(t, badResult.TokenValid)
}
