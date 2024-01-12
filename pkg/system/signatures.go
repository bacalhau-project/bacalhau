package system

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/config"
)

const (
	sigHash = crypto.SHA256 // hash function to use for sign/verify
)

// SignForClient signs a message with the user's private ID key.
// NOTE: must be called after InitConfig() or system will panic.
func SignForClient(msg []byte) (string, error) {
	privKey, err := config.GetClientPrivateKey()
	if err != nil {
		return "", err
	}

	return Sign(msg, privKey)
}

func Sign(msg []byte, privKey *rsa.PrivateKey) (string, error) {
	hash := sigHash.New()
	hash.Write(msg)
	hashBytes := hash.Sum(nil)

	sig, err := rsa.SignPKCS1v15(rand.Reader, privKey, sigHash, hashBytes)
	if err != nil {
		return "", fmt.Errorf("failed to sign message: %w", err)
	}

	return base64.StdEncoding.EncodeToString(sig), nil
}

// VerifyForClient verifies a signed message with the user's public ID key.
// NOTE: must be called after InitConfig() or system will panic.
func VerifyForClient(msg []byte, sig string) (bool, error) {
	pubKey, err := config.GetClientPublicKey()
	if err != nil {
		return false, err
	}

	hash := sigHash.New()
	hash.Write(msg)
	hashBytes := hash.Sum(nil)

	sigBytes, err := base64.StdEncoding.DecodeString(sig)
	if err != nil {
		return false, fmt.Errorf("failed to decode signature: %w", err)
	}

	// A successful verification is indicated by a nil return:
	return rsa.VerifyPKCS1v15(pubKey, sigHash, hashBytes, sigBytes) == nil, nil
}

// Verify verifies a signed message with the given encoding of a public key.
// Returns non nil if the key is invalid.
func Verify(msg []byte, sig, publicKey string) error {
	key, err := DecodePublicKey(publicKey)
	if err != nil {
		return fmt.Errorf("failed to decode public key: %w", err)
	}

	hash := sigHash.New()
	hash.Write(msg)
	hashBytes := hash.Sum(nil)

	sigBytes, err := base64.StdEncoding.DecodeString(sig)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// A successful verification is indicated by a nil return
	return rsa.VerifyPKCS1v15(key, sigHash, hashBytes, sigBytes)
}

// GetClientID returns a hash identifying a user based on their ID key.
// NOTE: must be called after InitConfig() or system will panic.
func GetClientID() string {
	clientID, err := config.GetClientID()
	if err != nil {
		panic(fmt.Sprintf("failed to load clientID: %s", err))
	}

	return clientID
}

// GetClientPublicKey returns a base64-encoding of the user's public ID key:
// NOTE: must be called after InitConfig() or system will panic.
func GetClientPublicKey() string {
	pkstr, err := config.GetClientPublicKeyString()
	if err != nil {
		panic(fmt.Sprintf("failed to load client public key: %s", err))
	}
	return pkstr
}

// PublicKeyMatchesID returns true if the given base64-encoded public key and
// the given client ID correspond to each other:
func PublicKeyMatchesID(publicKey, clientID string) (bool, error) {
	pkey, err := DecodePublicKey(publicKey)
	if err != nil {
		return false, fmt.Errorf("failed to decode public key: %w", err)
	}

	return clientID == ConvertToClientID(pkey), nil
}

// ConvertToClientID converts a public key to a client ID:
func ConvertToClientID(key *rsa.PublicKey) string {
	hash := sigHash.New()
	hash.Write(key.N.Bytes())
	hashBytes := hash.Sum(nil)

	return fmt.Sprintf("%x", hashBytes)
}

// DecodePublicKey decodes a public key from a string:
func DecodePublicKey(key string) (*rsa.PublicKey, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}

	return x509.ParsePKCS1PublicKey(keyBytes)
}
