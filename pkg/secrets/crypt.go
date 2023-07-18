package secrets

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/libp2p/go-libp2p/core/crypto"
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

func P2PPubKeyToRSA(pubKey crypto.PubKey) (*rsa.PublicKey, error) {
	key, err := crypto.PubKeyToStdKey(pubKey)
	if err != nil {
		return nil, err
	}

	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("unsupported key type")
	}

	return rsaKey, nil
}

func P2PPrivKeyToRSA(privKey crypto.PrivKey) (*rsa.PrivateKey, error) {
	key, err := crypto.PrivKeyToStdKey(privKey)
	if err != nil {
		return nil, err
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("unsupported key type")
	}

	return rsaKey, nil
}

func DecryptEnvStrings(currentEnv []string, decryptKey crypto.PrivKey) ([]string, error) {
	if len(currentEnv) == 0 {
		return currentEnv, nil
	}

	envs := make([]string, len(currentEnv))

	decryptionKey, _ := P2PPrivKeyToRSA(decryptKey) // requester node private key
	for _, kv := range currentEnv {
		if kv == "" {
			continue
		}
		parts := strings.Split(kv, "=")
		key, value := parts[0], parts[1]

		if !strings.HasPrefix(value, "ENC[") {
			envs = append(envs, kv)
			continue
		}

		value = value[4:]
		value = value[:len(value)-1]

		valueBytes, err := hex.DecodeString(value)
		if err != nil {
			return nil, err
		}

		decrypted, err := Decrypt(valueBytes, decryptionKey)
		if err != nil {
			return nil, err
		}

		envs = append(envs, fmt.Sprintf("%s=%s", key, string(decrypted)))
	}

	return envs, nil
}

func RecryptEnvString(currentEnv []string, decryptKey crypto.PrivKey, encryptKey crypto.PubKey) ([]string, error) {
	if len(currentEnv) == 0 {
		return currentEnv, nil
	}

	encryptionKey, _ := P2PPubKeyToRSA(encryptKey)  // compute node public key
	decryptionKey, _ := P2PPrivKeyToRSA(decryptKey) // requester node private key

	envs := make([]string, len(currentEnv))

	for _, kv := range currentEnv {
		if kv == "" {
			continue
		}
		parts := strings.Split(kv, "=")
		key, value := parts[0], parts[1]

		if !strings.HasPrefix(value, "ENC[") {
			envs = append(envs, kv)
			continue
		}

		value = value[4:]
		value = value[:len(value)-1]

		// fmt.Println("~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~")
		// fmt.Println(value)
		// fmt.Println(decryptionKey.Validate())
		// fmt.Println("~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~")

		valueBytes, err := hex.DecodeString(value)
		if err != nil {
			return nil, err
		}

		decrypted, err := Decrypt(valueBytes, decryptionKey)
		if err != nil {
			return nil, err
		}

		// Re-encrypt
		encryptedBytes, err := Encrypt(decrypted, encryptionKey)
		if err != nil {
			return nil, err
		}

		encryptedString := hex.EncodeToString(encryptedBytes)
		envs = append(envs, fmt.Sprintf("%s=ENC[%s]", key, encryptedString))
	}

	return envs, nil
}
