package config

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"os"

	libp2p_crypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	baccrypto "github.com/bacalhau-project/bacalhau/pkg/lib/crypto"
)

// GetClientPublicKeyString returns a base64-encoding of the user's public ID key:
// NOTE: must be called after InitConfig() or system will panic.
func GetClientPublicKeyString() (string, error) {
	userIDKey, err := loadUserIDKey()
	if err != nil {
		return "", err
	}

	return encodePublicKey(&userIDKey.PublicKey), nil
}

func GetClientPublicKey() (*rsa.PublicKey, error) {
	privKey, err := loadUserIDKey()
	if err != nil {
		return nil, err
	}
	return &privKey.PublicKey, nil
}

func GetClientPrivateKey() (*rsa.PrivateKey, error) {
	privKey, err := loadUserIDKey()
	if err != nil {
		return nil, err
	}
	return privKey, nil
}

func GetClientID() (string, error) {
	return loadClientID()
}

func GetInstallationUserID() (string, error) {
	return loadInstallationUserIDKey()
}

// loadClientID loads a hash identifying a user based on their ID key.
func loadClientID() (string, error) {
	key, err := loadUserIDKey()
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

// encodePublicKey encodes a public key as a string:
func encodePublicKey(key *rsa.PublicKey) string {
	return base64.StdEncoding.EncodeToString(x509.MarshalPKCS1PublicKey(key))
}

func loadInstallationUserIDKey() (string, error) {
	key := viper.GetString(types.UserInstallationID)
	if key == "" {
		return "", fmt.Errorf("config error: user-installation-id-key not set")
	}
	return key, nil
}

// loadUserIDKey loads the user ID key from whatever source is configured.
func loadUserIDKey() (*rsa.PrivateKey, error) {
	keyFile := viper.GetString(types.UserKeyPath)
	if keyFile == "" {
		return nil, fmt.Errorf("config error: user-id-key not set")
	}

	return baccrypto.LoadPKCS1KeyFile(keyFile)
}

func GetLibp2pPrivKey() (libp2p_crypto.PrivKey, error) {
	return loadLibp2pPrivKey()
}

func loadLibp2pPrivKey() (libp2p_crypto.PrivKey, error) {
	keyFile := viper.GetString(types.UserLibp2pKeyPath)
	if keyFile == "" {
		return nil, fmt.Errorf("config error: libp2p private key not set")
	}

	keyBytes, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %v", err)
	}
	// base64 decode keyBytes
	b64, err := base64.StdEncoding.DecodeString(string(keyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %v", err)
	}
	// parse the private key
	key, err := libp2p_crypto.UnmarshalPrivateKey(b64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}
	return key, nil
}
