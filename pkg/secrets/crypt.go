package secrets

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
)

var hasher = sha256.New()

func Encrypt(data []byte, publicKey *rsa.PublicKey) ([]byte, error) {
	encrypted, err := rsa.EncryptOAEP(hasher, rand.Reader, publicKey, data, nil)
	if err != nil {
		return nil, err
	}

	return encrypted, nil
}

func Decrypt(encryptedData []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
	decrypted, err := rsa.DecryptOAEP(hasher, rand.Reader, privateKey, encryptedData, nil)
	if err != nil {
		return nil, err
	}

	return decrypted, nil
}

func EncryptEnv(env map[string]string, publicKey *rsa.PublicKey) (map[string]string, error) {
	encryptedMap := make(map[string]string)

	for k, v := range env {
		encryptedBytes, err := Encrypt([]byte(v), publicKey)
		if err != nil {
			return nil, err
		}

		encryptedMap[k] = hex.EncodeToString(encryptedBytes)
	}

	return encryptedMap, nil
}

func DecryptEnv(encryptedMap map[string]string, privateKey *rsa.PrivateKey) (map[string]string, error) {
	decryptedMap := make(map[string]string)

	for k, v := range encryptedMap {
		valueAsBytes, err := hex.DecodeString(v)
		if err != nil {
			return nil, err
		}

		decryptedBytes, err := Decrypt(valueAsBytes, privateKey)
		if err != nil {
			return nil, err
		}

		decryptedMap[k] = string(decryptedBytes)
	}

	return decryptedMap, nil
}
