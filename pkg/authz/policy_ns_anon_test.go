//go:build unit || !integration

package authz

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/lib/policy"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const (
	NamespaceNoPermission uint8 = 0b0000
	NamespaceReadable     uint8 = 0b0001
	NamespaceWritable     uint8 = 0b0010
	NamespaceDownloadable uint8 = 0b0100
	NamespaceCancellable  uint8 = 0b1000
)

func getJWTWithNamespace(t *testing.T, namespace string, perms uint8) string {
	token, err := jwt.NewWithClaims(jwt.GetSigningMethod("none"), jwt.MapClaims{
		"ns": map[string]uint8{namespace: perms},
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)
	return token
}

func TestAppliesAnonymousNamespacePolicy(t *testing.T) {
	cases := []struct {
		name            string
		job_namespace   string
		client_id       string
		token_namespace string
		token_perms     uint8
		method          string
		path            string
		checker         func(require.TestingT, bool, ...interface{})
	}{
		{"allow valid job submission",
			"test", "test", "test", NamespaceWritable, http.MethodPut, "/api/v1/orchestrator/jobs", require.True},
		{"deny job submit to unwritable namespace",
			"test", "test", "test", NamespaceReadable, http.MethodPut, "/api/v1/orchestrator/jobs", require.False},
		{"deny job submit to alternative namespace",
			"other", "other", "test", NamespaceWritable, http.MethodPut, "/api/v1/orchestrator/jobs", require.False},
		{"allow valid job read",
			"test", "test", "test", NamespaceReadable, http.MethodGet, "/api/v1/orchestrator/jobs", require.True},
		{"deny job read to unreadable namespace",
			"test", "test", "test", NamespaceWritable, http.MethodGet, "/api/v1/orchestrator/jobs", require.False},
		{"deny job read to alternative namespace",
			"other", "other", "test", NamespaceReadable, http.MethodGet, "/api/v1/orchestrator/jobs", require.False},
		{"allow reading other APIs without token",
			"other", "other", "test", NamespaceNoPermission, http.MethodGet, "/api/v1/orchestrator/nodes", require.True},
		{"deny writing other APIs",
			"other", "other", "test", NamespaceNoPermission, http.MethodDelete, "/api/v1/orchestrator/nodes", require.False},
	}

	policy, err := policy.FromFS(policies, "policies/policy_ns_anon.rego")
	require.NoError(t, err)
	authorizer := NewPolicyAuthorizer(policy)

	for _, testcase := range cases {
		t.Run(testcase.name, func(t *testing.T) {
			logger.ConfigureTestLogging(t)

			job := &models.Job{
				Namespace: testcase.job_namespace,
				Meta: map[string]string{
					"bacalhau.org/client.id": testcase.client_id,
				},
			}

			body, err := yaml.Marshal(job)
			require.NoError(t, err)

			request, err := http.NewRequest(testcase.method, testcase.path, bytes.NewReader(body))
			require.NoError(t, err)

			if testcase.token_perms > 0 {
				token := getJWTWithNamespace(t, testcase.token_namespace, testcase.token_perms)
				request.Header.Add("Authorization", "Bearer "+token)
			}

			result, err := authorizer.Authorize(request)
			require.NoError(t, err)
			testcase.checker(t, result.Approved)
		})
	}
}
