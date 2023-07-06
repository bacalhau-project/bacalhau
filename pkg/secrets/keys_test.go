//go:build unit || !integration

package secrets_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/secrets"
	"github.com/stretchr/testify/suite"
)

type KeySuite struct {
	suite.Suite

	tmpFolder string
}

func (s *KeySuite) SetupSuite() {
	s.tmpFolder, _ = os.MkdirTemp("", "bacalhau-key-test")
}

func (s *KeySuite) TearDownSuite() {
	os.RemoveAll(s.tmpFolder)
}

func TestKeySuite(t *testing.T) {
	suite.Run(t, new(KeySuite))
}

func (s *KeySuite) TestNewKeys() {
	suffix := "test-new-keys"

	privKeyPath := filepath.Join(s.tmpFolder, fmt.Sprintf("secrets-priv-%s.pem", suffix))
	s.NoFileExists(privKeyPath)

	pubKeyPath := filepath.Join(s.tmpFolder, fmt.Sprintf("secrets-pub-%s.pem", suffix))
	s.NoFileExists(pubKeyPath)

	priv, pub, err := secrets.GetSecretsKeyPair(s.tmpFolder, suffix)
	s.NoError(err)
	s.NotNil(priv)
	s.NotNil(pub)
	s.FileExists(privKeyPath)
	s.FileExists(pubKeyPath)

	privLoaded, pubLoaded, err := secrets.GetSecretsKeyPair(s.tmpFolder, suffix)
	s.NoError(err)
	s.True(priv.Equal(privLoaded))
	s.True(pub.Equal(pubLoaded))
}

func (s *KeySuite) TestKeyConversion() {
	suffix := "test-key-conversion"
	_, pub, err := secrets.GetSecretsKeyPair(s.tmpFolder, suffix)
	s.NoError(err)

	pubBytes := secrets.PublicKeyToBytes(pub)
	pubRoundtripped, err := secrets.BytesToPublicKey(pubBytes)
	s.NoError(err)
	s.True(pub.Equal(pubRoundtripped))
}
