package secrets

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const BitsForSecretsKeyPair = 4096

func GetSecretsKeyPair(folder string, suffix string) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKeyPath := KeyPath(folder, "priv", suffix)
	publicKeyPath := KeyPath(folder, "pub", suffix)

	var err error
	var privatekey *rsa.PrivateKey
	var publickey *rsa.PublicKey

	if _, err = os.Stat(privateKeyPath); errors.Is(err, os.ErrNotExist) {
		privatekey, err = rsa.GenerateKey(rand.Reader, BitsForSecretsKeyPair)
		if err != nil {
			return nil, nil, err
		}
		publickey = &privatekey.PublicKey

		privateKeyBytes := x509.MarshalPKCS1PrivateKey(privatekey)
		err = keyToFile(privateKeyBytes, "RSA PRIVATE KEY", privateKeyPath)
		if err != nil {
			return nil, nil, err
		}

		publicKeyBytes := x509.MarshalPKCS1PublicKey(publickey)
		err = keyToFile(publicKeyBytes, "RSA PUBLIC KEY", publicKeyPath)
		if err != nil {
			return nil, nil, err
		}

		return privatekey, publickey, nil
	}

	privateBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, nil, err
	}

	block, _ := pem.Decode(privateBytes)
	privatekey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, nil, err
	}

	publicBytes, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, nil, err
	}
	block, _ = pem.Decode(publicBytes)
	publickey, err = x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, nil, err
	}

	return privatekey, publickey, nil
}

// PublicKeyToBytes marshals the provided publickey into bytes so that
// it can be transmitted as part of the node info.
func PublicKeyToBytes(publicKey *rsa.PublicKey) []byte {
	return x509.MarshalPKCS1PublicKey(publicKey)
}

// BytesToPublicKey converts the provided byte slice into a public key
// useful for decoding the public key from the node info
func BytesToPublicKey(transmittedBytes []byte) (*rsa.PublicKey, error) {
	return x509.ParsePKCS1PublicKey(transmittedBytes)
}

func KeyPath(folder string, vis string, suffix string) string {
	return filepath.Join(folder, fmt.Sprintf("secrets-%s-%s.pem", vis, suffix))
}

func keyToFile(keyBytes []byte, label string, path string) error {
	keyBlock := &pem.Block{
		Type:  label,
		Bytes: keyBytes,
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	err = pem.Encode(file, keyBlock)
	if err != nil {
		return err
	}

	return nil
}
