package crypto

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"os"

	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
	"github.com/pkg/errors"
)

func LoadPKCS1KeyFile(keyFile string) (*rsa.PrivateKey, error) {
	file, err := os.Open(keyFile)
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
