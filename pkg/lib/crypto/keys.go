package crypto

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"os"

	"github.com/pkg/errors"

	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
)

type UserKey struct {
	sk      *rsa.PrivateKey
	sigHash crypto.Hash
}

func (u *UserKey) PrivateKey() *rsa.PrivateKey {
	return u.sk
}

func (u *UserKey) PublicKey() *rsa.PublicKey {
	return &u.sk.PublicKey
}

func (u *UserKey) ClientID() string {
	hash := u.sigHash.New()
	hash.Write(u.sk.N.Bytes())
	hashBytes := hash.Sum(nil)

	return fmt.Sprintf("%x", hashBytes)
}

func LoadUserKey(path string) (*UserKey, error) {
	sk, err := LoadPKCS1KeyFile(path)
	if err != nil {
		return nil, err
	}
	return &UserKey{
		sk:      sk,
		sigHash: crypto.SHA256,
	}, nil
}

func LoadPKCS1KeyFile(keyFile string) (*rsa.PrivateKey, error) {
	file, err := os.Open(keyFile) //nolint:gosec // G304: Caller responsible for validating key file path
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open key file %q", keyFile)
	}
	defer closer.CloseWithLogOnError(keyFile, file)
	return LoadPKCS1Key(file)
}

func LoadPKCS1Key(in io.Reader) (*rsa.PrivateKey, error) {
	keyBytes, err := io.ReadAll(in)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read key")
	}

	keyBlock, _ := pem.Decode(keyBytes)
	if keyBlock == nil {
		return nil, errors.Wrap(err, "failed to decode key")
	}

	// TODO: #3159 Add support for both rsa _and_ ecdsa private keys, see crypto.PrivateKey.
	//       Since we have access to the private key we can hack it by signing a
	//       message twice and comparing them, rather than verifying directly.
	// ecdsaKey, err = x509.ParseECPrivateKey(keyBlock.Bytes)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to parse user: %w", err)
	// }

	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse key")
	}

	return key, nil
}
