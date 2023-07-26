//go:build unit || !integration

package secrets_test

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/secrets"
	"github.com/stretchr/testify/require"
)

func TestEncryptedValue(t *testing.T) {
	msg := "hello world"

	pkey, _ := rsa.GenerateKey(rand.Reader, 2048)
	pubKey := pkey.PublicKey

	encrypted, err := secrets.EncryptData(&pubKey, "mykey", msg)
	require.NoError(t, err)

	encryptedString := encrypted.String()
	require.Contains(t, encryptedString, "ENC[")

	decrypted, err := secrets.DecryptData(pkey, "mykey", encryptedString)
	require.NoError(t, err)
	require.Equal(t, msg, decrypted)
}
