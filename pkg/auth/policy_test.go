//go:build unit || !integration

package auth

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAllowsWhenPolicySaysAllow(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "/api/v1/hello", nil)
	require.NoError(t, err)

	authorizer, err := NewPolicyAuthorizer(policies, "policies/policy_test_allow.rego")
	require.NoError(t, err)
	require.NotNil(t, authorizer)

	result, err := authorizer.ShouldAllow(request)
	require.NoError(t, err)
	require.True(t, result.Approved)
}

func TestDeniesWhenPolicySaysDeny(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "/api/v1/hello", nil)
	require.NoError(t, err)

	authorizer, err := NewPolicyAuthorizer(policies, "policies/policy_test_deny.rego")
	require.NoError(t, err)
	require.NotNil(t, authorizer)

	result, err := authorizer.ShouldAllow(request)
	require.NoError(t, err)
	require.False(t, result.Approved)
}

func TestPolicyEvaluatedAgainstHTTPRequest(t *testing.T) {
	authorizer, err := NewPolicyAuthorizer(policies, "policies/policy_test_http.rego")
	require.NoError(t, err)
	require.NotNil(t, authorizer)

	goodRequest, err := http.NewRequest(http.MethodGet, "/api/v1/hello", nil)
	require.NoError(t, err)

	goodResult, err := authorizer.ShouldAllow(goodRequest)
	require.NoError(t, err)
	require.True(t, goodResult.Approved)

	badRequest, err := http.NewRequest(http.MethodDelete, "/api/v1/hello", nil)
	require.NoError(t, err)

	badResult, err := authorizer.ShouldAllow(badRequest)
	require.NoError(t, err)
	require.False(t, badResult.Approved)
}
