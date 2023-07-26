package secrets

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

var hasher = sha256.New()

// EncryptData will encrypt `data` using the provided PublicKey to return
// an EncryptedValue which can be encoded as a string for transport in a
// job spec
func EncryptData(key *rsa.PublicKey, keyID string, data string) (*EncryptedValue, error) {
	encryptedBytes, err := rsa.EncryptOAEP(hasher, rand.Reader, key, []byte(data), nil)
	if err != nil {
		return nil, err
	}

	return &EncryptedValue{
		KeyID: keyID,
		Data:  hex.EncodeToString(encryptedBytes),
	}, nil
}

func DecryptData(key *rsa.PrivateKey, keyID string, encVal string) (string, error) {
	enc, err := ParseEncryptedValue(encVal)
	if err != nil {
		return "", err
	}

	if keyID != enc.KeyID {
		return "", fmt.Errorf("unexpected key id, expected (%s) got(%s)", enc.KeyID, keyID)
	}

	// decode hex string
	unhexed, err := hex.DecodeString(enc.Data)
	if err != nil {
		return "", err
	}

	decryptedBytes, err := rsa.DecryptOAEP(hasher, rand.Reader, key, unhexed, nil)
	if err != nil {
		return "", err
	}

	return string(decryptedBytes), nil
}

func RecryptData(
	decryptionKey *rsa.PrivateKey,
	originalKeyID string,
	encryptionKey *rsa.PublicKey,
	newKeyID string,
	encryptedString string) (*EncryptedValue, error) {
	decrypted, err := DecryptData(decryptionKey, originalKeyID, encryptedString)
	if err != nil {
		return nil, err
	}

	encVal, err := EncryptData(encryptionKey, newKeyID, decrypted)
	if err != nil {
		return nil, err
	}

	return encVal, nil
}

func JobNeedsDecrypting(spec model.Spec) bool {
	if spec.Engine == model.EngineDocker {
		if len(spec.Docker.EnvironmentVariables) == 0 {
			return false
		}

		// Do any of the values contain the ENC[] wrapper
		for _, kv := range spec.Docker.EnvironmentVariables {
			_, v := splitEnv(kv)
			if !strings.HasPrefix(v, "ENC[") {
				return false
			}
		}

		return len(spec.Docker.EnvironmentVariables) > 0
	}
	return false
}

func DecryptEnv(spec model.Spec, privKey *rsa.PrivateKey, keyID string) (model.Spec, error) {
	decryptedEnv := make([]string, 0, len(spec.Docker.EnvironmentVariables))

	for _, kv := range spec.Docker.EnvironmentVariables {
		k, v := splitEnv(kv)

		data, err := DecryptData(privKey, keyID, v)
		if err != nil {
			return model.Spec{}, err
		}

		decryptedEnv = append(decryptedEnv, fmt.Sprintf("%s=%s", k, data))
	}

	spec.Docker.EnvironmentVariables = decryptedEnv
	return spec, nil
}

func RecryptEnv(spec model.Spec,
	requesterPrivateKey *rsa.PrivateKey, requesterKeyID string,
	computePublicKey *rsa.PublicKey, computeKeyID string) (model.Spec, error) {
	recrypted := make([]string, 0, len(spec.Docker.EnvironmentVariables))

	for _, kv := range spec.Docker.EnvironmentVariables {
		k, v := splitEnv(kv)

		ev, err := RecryptData(requesterPrivateKey, requesterKeyID, computePublicKey, computeKeyID, v)
		if err != nil {
			return model.Spec{}, err
		}

		recrypted = append(recrypted, fmt.Sprintf("%s=%s", k, ev.String()))
	}

	spec.Docker.EnvironmentVariables = recrypted
	return spec, nil
}

func splitEnv(s string) (k string, v string) {
	parts := strings.Split(s, "=")
	return parts[0], parts[1]
}
