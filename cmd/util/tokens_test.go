//go:build unit || !integration

package util

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
)

const (
	testDomain          = "http://example.com:1234"
	otherDomain         = "http://examples.com:1234"
	testDomainOtherPort = "http://example.com:1235"
)

var exampleToken = apimodels.HTTPCredential{
	Scheme: "Bearer",
	Value:  "some-token",
}

func TestReadingEmptyTokensFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tokens.json")

	token, err := ReadToken(path, testDomain)
	require.NoError(t, err)
	require.Nil(t, token)
}

func TestTokenRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tokens.json")

	err := WriteToken(path, testDomain, &exampleToken)
	require.NoError(t, err)

	t.Run("read back same domain", func(t *testing.T) {
		token, err := ReadToken(path, testDomain)
		require.NoError(t, err)
		require.Equal(t, exampleToken.String(), token.String())
	})

	t.Run("read other domain", func(t *testing.T) {
		token, err := ReadToken(path, otherDomain)
		require.NoError(t, err)
		require.Nil(t, token)
	})

	t.Run("read other port", func(t *testing.T) {
		token, err := ReadToken(path, testDomainOtherPort)
		require.NoError(t, err)
		require.Nil(t, token)
	})
}

func TestReadTokenFromEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tokens.json")

	err := os.WriteFile(path, []byte{}, util.OS_USER_RW)
	require.NoError(t, err)

	token, err := ReadToken(path, testDomain)
	require.NoError(t, err)
	require.Nil(t, token)
}

func TestWriteTokenToEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tokens.json")

	err := os.WriteFile(path, []byte{}, util.OS_USER_RW)
	require.NoError(t, err)

	err = WriteToken(path, testDomain, &exampleToken)
	require.NoError(t, err)
}

func TestWriteNilTokenIsValid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tokens.json")

	err := WriteToken(path, testDomain, nil)
	require.NoError(t, err)

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.NotZero(t, info.Size())

	token, err := ReadToken(path, testDomain)
	require.NoError(t, err)
	require.Nil(t, token)
}

func TestWriteNilTokenDeletesToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tokens.json")

	err := WriteToken(path, testDomain, &exampleToken)
	require.NoError(t, err)

	token, err := ReadToken(path, testDomain)
	require.NoError(t, err)
	require.NotNil(t, token)

	err = WriteToken(path, testDomain, nil)
	require.NoError(t, err)

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.NotZero(t, info.Size())

	token, err = ReadToken(path, testDomain)
	require.NoError(t, err)
	require.Nil(t, token)
}
