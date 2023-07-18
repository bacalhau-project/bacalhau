//go:build unit || !integration

package secrets_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"os"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/secrets"
	"github.com/stretchr/testify/suite"
)

type CryptSuite struct {
	suite.Suite

	tmpFolder string
}

func (s *CryptSuite) SetupSuite() {
	s.tmpFolder, _ = os.MkdirTemp("", "bacalhau-key-test")
}

func (s *CryptSuite) TearDownSuite() {
	os.RemoveAll(s.tmpFolder)
}

func TestCryptSuite(t *testing.T) {
	suite.Run(t, new(CryptSuite))
}

const BitsForSecretsKeyPair = 2048

func getSecretsKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	var err error
	var privatekey *rsa.PrivateKey
	var publickey *rsa.PublicKey

	privatekey, err = rsa.GenerateKey(rand.Reader, BitsForSecretsKeyPair)
	if err != nil {
		return nil, nil, err
	}
	publickey = &privatekey.PublicKey

	return privatekey, publickey, nil
}

func (s *CryptSuite) TestEncryptDecryptCycle() {
	priv, pub, err := getSecretsKeyPair()
	s.NoError(err)

	message := "this is a string to be encrypted"

	encrypted, err := secrets.Encrypt([]byte(message), pub)
	s.NoError(err)
	s.NotNil(encrypted)

	// Decrypt without labels
	decrypted, err := secrets.Decrypt(encrypted, priv)
	s.NoError(err)
	s.NotNil(decrypted)

	decryptedString := string(decrypted)
	s.Equal(message, decryptedString)
}

func (s *CryptSuite) TestEncryptDecryptEnv() {
	priv, pub, err := getSecretsKeyPair()
	s.NoError(err)

	env := make(map[string]string)
	env["MESSAGE"] = "A value we want encrypted"
	env["RANDOM_KEY"] = "Another value"
	env["API_KEY"] = "apikey"

	encMap, err := secrets.EncryptEnv(env, pub)
	s.NoError(err)

	decMap, err := secrets.DecryptEnv(encMap, priv)
	s.NoError(err)
	s.Equal(decMap, env)
}

func (s *CryptSuite) TestEncryptDecryptEnvFailWrongPrivKey() {
	_, pub, err := getSecretsKeyPair()
	s.NoError(err)

	wrongPriv, _, err := getSecretsKeyPair()
	s.NoError(err)

	env := make(map[string]string)
	env["MESSAGE"] = "A value we want encrypted"
	env["RANDOM_KEY"] = "Another value"
	env["API_KEY"] = "apikey"

	encMap, err := secrets.EncryptEnv(env, pub)
	s.NoError(err)

	decMap, err := secrets.DecryptEnv(encMap, wrongPriv)
	s.Error(err)
	s.Nil(decMap)
}

func (s *CryptSuite) TestEncryptDecryptEnvFailNotEncrypted() {
	priv, _, err := getSecretsKeyPair()
	s.NoError(err)

	env := make(map[string]string)
	env["MESSAGE"] = "A value we want encrypted"
	env["RANDOM_KEY"] = "Another value"
	env["API_KEY"] = "apikey"

	// Invalid hex byte
	decMap, err := secrets.DecryptEnv(env, priv)
	s.Error(err)
	s.Nil(decMap)

	// decryption error
	env["API_KEY"] = hex.EncodeToString([]byte(env["API_KEY"]))
	decMap, err = secrets.DecryptEnv(env, priv)
	s.Error(err)
	s.Nil(decMap)
}
