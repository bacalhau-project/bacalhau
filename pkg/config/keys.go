package config

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"os"

	libp2p_crypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
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

// loadUserIDKey loads the user ID key from whatever source is configured.
func loadUserIDKey() (*rsa.PrivateKey, error) {
	keyFile := viper.GetString(NodeUserUserKeyPath)
	if keyFile == "" {
		return nil, fmt.Errorf("config error: user-id-key not set")
	}

	file, err := os.Open(keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open user ID key file: %w", err)
	}
	defer closer.CloseWithLogOnError("user ID key file", file)

	keyBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read user ID key file: %w", err)
	}

	keyBlock, _ := pem.Decode(keyBytes)
	if keyBlock == nil {
		return nil, fmt.Errorf("failed to decode user ID key file %q", keyFile)
	}

	// TODO: Add support for both rsa _and_ ecdsa private keys, see cryto.PrivateKey.
	//       Since we have access to the private key we can hack it by signing a
	//       message twice and comparing them, rather than verifying directly.
	// ecdsaKey, err = x509.ParseECPrivateKey(keyBlock.Bytes)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to parse user: %w", err)
	// }

	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user ID key file: %w", err)
	}

	return key, nil
}

func GetLibp2pPrivKey() (libp2p_crypto.PrivKey, error) {
	return loadLibp2pPrivKey()
}

func loadLibp2pPrivKey() (libp2p_crypto.PrivKey, error) {
	keyFile := viper.GetString(NodeUserLibp2pKeyPath)
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
