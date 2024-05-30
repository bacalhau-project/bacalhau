package system

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
)

type Signer interface {
	// Sign signs a message with the user's private ID key.
	Sign(msg []byte) (string, error)
	PublicKey() *rsa.PublicKey
	PublicKeyString() string
}

func NewMessageSigner(sk *rsa.PrivateKey) *MessageSigner {
	return &MessageSigner{sk: sk}
}

type MessageSigner struct {
	sk *rsa.PrivateKey
}

func (m *MessageSigner) PublicKey() *rsa.PublicKey {
	return &m.sk.PublicKey
}

// PublicKeyString returns a base64-encoding of the user's public ID key:
func (m *MessageSigner) PublicKeyString() string {
	pk := m.PublicKey()
	return encodePublicKey(pk)
}

func (m *MessageSigner) Sign(msg []byte) (string, error) {
	hash := sigHash.New()
	hash.Write(msg)
	hashBytes := hash.Sum(nil)

	sig, err := rsa.SignPKCS1v15(rand.Reader, m.sk, sigHash, hashBytes)
	if err != nil {
		return "", fmt.Errorf("failed to sign message: %w", err)
	}

	return base64.StdEncoding.EncodeToString(sig), nil
}

const (
	sigHash = crypto.SHA256 // hash function to use for sign/verify
)

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

// encodePublicKey encodes a public key as a string:
func encodePublicKey(key *rsa.PublicKey) string {
	return base64.StdEncoding.EncodeToString(x509.MarshalPKCS1PublicKey(key))
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
