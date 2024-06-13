//go:build unit || !integration

package challenge

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/authn"
	"github.com/bacalhau-project/bacalhau/pkg/lib/policy"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func setup(t *testing.T) authn.Authenticator {
	logger.ConfigureTestLogging(t)

	anonPolicy, err := (policy.FromFS(policies, "challenge_ns_anon.rego"))
	require.NoError(t, err)

	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	return NewAuthenticator(anonPolicy, NewStringMarshaller("test"), rsaKey, "node")
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
	require.Equal(t, authn.MethodTypeChallenge, requirement.Type)
	require.NoError(t, json.Unmarshal(*requirement.Params, &request{}))
}

func TestBadlyStructuredChallenge(t *testing.T) {
	authenticator := setup(t)
	response := response{
		PhraseSignature: "blagging it",
		PublicKey:       "bad key",
	}

	auth := try(t, authenticator, response)
	require.False(t, auth.Success)
	require.Empty(t, auth.Token)
}

func TestBadlySignedChallenge(t *testing.T) {
	authenticator := setup(t)
	userPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	signer := system.NewMessageSigner(userPrivKey)
	userPubKey := base64.StdEncoding.EncodeToString(x509.MarshalPKCS1PublicKey(&userPrivKey.PublicKey))

	signature, err := signer.Sign([]byte("other input phrase"))
	require.NoError(t, err)
	response := response{
		PhraseSignature: signature,
		PublicKey:       userPubKey,
	}

	auth := try(t, authenticator, response)
	require.False(t, auth.Success)
	require.Empty(t, auth.Token)
}

func TestGoodChallenge(t *testing.T) {
	authenticator := setup(t)

	var req request
	err := json.Unmarshal(*authenticator.Requirement().Params, &req)
	require.NoError(t, err)

	userPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	signer := system.NewMessageSigner(userPrivKey)

	userPubKey := base64.StdEncoding.EncodeToString(x509.MarshalPKCS1PublicKey(&userPrivKey.PublicKey))
	require.NoError(t, err)

	signature, err := signer.Sign(req.InputPhrase)
	require.NoError(t, err)

	response := response{
		PhraseSignature: signature,
		PublicKey:       userPubKey,
	}

	auth := try(t, authenticator, response)
	require.True(t, auth.Success, auth.Reason)
	require.NotEmpty(t, auth.Token)
}
