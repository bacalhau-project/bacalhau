//go:build unit || !integration

package challenge

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/authn"
	"github.com/bacalhau-project/bacalhau/pkg/lib/policy"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
)

type testData []byte

func (t testData) MarshalBinary() ([]byte, error) {
	return t, nil
}

func setup(t *testing.T) authn.Authenticator {
	logger.ConfigureTestLogging(t)

	anonPolicy, err := (policy.FromFS(policies, "challenge_ns_anon.rego"))
	require.NoError(t, err)

	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	return NewAuthenticator(anonPolicy, testData([]byte("test")), rsaKey, "node")
}

func try(t *testing.T, authenticator authn.Authenticator, r response) authn.Authentication {
	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(r)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "/api/v1/auth/challenge", body)
	require.NoError(t, err)

	auth, err := authenticator.Authenticate(req)
	require.NoError(t, err)
	return auth
}

func TestRequirement(t *testing.T) {
	authenticator := setup(t)

	requirement := authenticator.Requirement()
	require.Equal(t, authn.MethodTypeChallenge, requirement.Type)
	require.IsType(t, request{}, requirement.Params)
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

	userPubKey := base64.StdEncoding.EncodeToString(x509.MarshalPKCS1PublicKey(&userPrivKey.PublicKey))

	signature, err := system.Sign([]byte("other input phrase"), userPrivKey)
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
	request := (authenticator.Requirement().Params).(request)

	userPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	userPubKey := base64.StdEncoding.EncodeToString(x509.MarshalPKCS1PublicKey(&userPrivKey.PublicKey))

	signature, err := system.Sign(request.InputPhrase, userPrivKey)
	response := response{
		PhraseSignature: signature,
		PublicKey:       userPubKey,
	}

	auth := try(t, authenticator, response)
	require.True(t, auth.Success, auth.Reason)
	require.NotEmpty(t, auth.Token)
}
