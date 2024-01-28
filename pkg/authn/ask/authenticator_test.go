//go:build unit || !integration

package ask

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"embed"
	"encoding/json"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/authn"
	"github.com/bacalhau-project/bacalhau/pkg/lib/policy"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/stretchr/testify/require"
)

//go:embed *.rego
var policies embed.FS

func setup(t *testing.T) authn.Authenticator {
	logger.ConfigureTestLogging(t)

	authPolicy, err := (policy.FromFS(policies, "ask_ns_test_password.rego"))
	require.NoError(t, err)

	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	return NewAuthenticator(authPolicy, rsaKey, "node")
}

func try(t *testing.T, authenticator authn.Authenticator, r any) authn.Authentication {
	req, err := json.Marshal(r)
	require.NoError(t, err)

	auth, err := authenticator.Authenticate(context.Background(), req)
	require.NoError(t, err)
	return auth
}

func TestRequirement(t *testing.T) {
	authenticator := setup(t)

	requirement := authenticator.Requirement()
	require.Equal(t, authn.MethodTypeAsk, requirement.Type)
	require.NoError(t, json.Unmarshal(*requirement.Params, &requiredSchema{}))
}

func TestUnknownUser(t *testing.T) {
	authenticator := setup(t)

	auth := try(t, authenticator, map[string]string{
		"username": "robert",
		"password": "password",
	})
	require.False(t, auth.Success, auth.Reason)
	require.Empty(t, auth.Token)
}

func TestIncorrectPassword(t *testing.T) {
	authenticator := setup(t)

	auth := try(t, authenticator, map[string]string{
		"username": "username",
		"password": "username",
	})
	require.False(t, auth.Success, auth.Reason)
	require.Empty(t, auth.Token)
}

func TestGoodResponse(t *testing.T) {
	authenticator := setup(t)

	auth := try(t, authenticator, map[string]string{
		"username": "username",
		"password": "password",
	})
	require.True(t, auth.Success, auth.Reason)
	require.NotEmpty(t, auth.Token)
}
