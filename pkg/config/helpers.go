package config

import (
	"crypto"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	libp2p_crypto "github.com/libp2p/go-libp2p/core/crypto"

	baccrypto "github.com/bacalhau-project/bacalhau/pkg/lib/crypto"
)

// KeyAsEnvVar returns the environment variable corresponding to a config key
func KeyAsEnvVar(key string) string {
	return strings.ToUpper(
		fmt.Sprintf("%s_%s", environmentVariablePrefix, environmentVariableReplace.Replace(key)),
	)
}

func GetClientID(path string) (string, error) {
	return loadClientID(path)
}

// loadClientID loads a hash identifying a user based on their ID key.
func loadClientID(path string) (string, error) {
	key, err := loadUserIDKey(path)
	if err != nil {
		return "", fmt.Errorf("failed to load user ID key: %w", err)
	}

	return convertToClientID(&key.PublicKey), nil
}

const (
	sigHash = crypto.SHA256 // hash function to use for sign/verify
)

// convertToClientID converts a public key to a client ID:
func convertToClientID(key *rsa.PublicKey) string {
	hash := sigHash.New()
	hash.Write(key.N.Bytes())
	hashBytes := hash.Sum(nil)

	return fmt.Sprintf("%x", hashBytes)
}

func DevstackGetShouldPrintInfo() bool {
	return os.Getenv("DEVSTACK_PRINT_INFO") != ""
}

func DevstackSetShouldPrintInfo() {
	os.Setenv("DEVSTACK_PRINT_INFO", "1")
}

func DevstackEnvFile() string {
	return os.Getenv("DEVSTACK_ENV_FILE")
}

func ShouldKeepStack() bool {
	return os.Getenv("KEEP_STACK") != ""
}

const (
	DockerUsernameEnvVar = "DOCKER_USERNAME"
	DockerPasswordEnvVar = "DOCKER_PASSWORD"
)

type DockerCredentials struct {
	Username string
	Password string
}

func (d *DockerCredentials) IsValid() bool {
	return d.Username != "" && d.Password != ""
}

func GetDockerCredentials() DockerCredentials {
	return DockerCredentials{
		Username: os.Getenv(DockerUsernameEnvVar),
		Password: os.Getenv(DockerPasswordEnvVar),
	}
}

// PreferredAddress will allow for the specifying of
// the preferred address to listen on for cases where it
// is not clear, or where the address does not appear when
// using 0.0.0.0
func PreferredAddress() string {
	return os.Getenv("BACALHAU_PREFERRED_ADDRESS")
}

func GetClientPrivateKey(path string) (*rsa.PrivateKey, error) {
	privKey, err := loadUserIDKey(path)
	if err != nil {
		return nil, err
	}
	return privKey, nil
}

// loadUserIDKey loads the user ID key from whatever source is configured.
func loadUserIDKey(path string) (*rsa.PrivateKey, error) {
	return baccrypto.LoadPKCS1KeyFile(path)
}

func GetLibp2pPrivKey(path string) (libp2p_crypto.PrivKey, error) {
	return loadLibp2pPrivKey(path)
}

func loadLibp2pPrivKey(path string) (libp2p_crypto.PrivKey, error) {
	keyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}
	// base64 decode keyBytes
	b64, err := base64.StdEncoding.DecodeString(string(keyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}
	// parse the private key
	key, err := libp2p_crypto.UnmarshalPrivateKey(b64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}
	return key, nil
}
