package verifier

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/libp2p/go-libp2p/core/crypto"
)

type Encrypter struct {
	privateKey crypto.PrivKey
}

func NewEncrypter(privateKey crypto.PrivKey) Encrypter {
	return Encrypter{
		privateKey: privateKey,
	}
}

func (e Encrypter) Encrypt(ctx context.Context, data, libp2pKeyBytes []byte) ([]byte, error) {
	_, span := system.NewSpan(ctx, system.GetTracer(), "pkg/verifier.Encrypter.Encrypt")
	defer span.End()

	unmarshalledPublicKey, err := crypto.UnmarshalPublicKey(libp2pKeyBytes)
	if err != nil {
		return nil, err
	}
	publicKeyBytes, err := unmarshalledPublicKey.Raw()
	if err != nil {
		return nil, err
	}
	genericPublicKey, err := x509.ParsePKIXPublicKey(publicKeyBytes)
	if err != nil {
		return nil, err
	}
	rsaPublicKey, ok := genericPublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("could not cast public key to RSA")
	}
	return rsa.EncryptOAEP(
		sha512.New(),
		rand.Reader,
		rsaPublicKey,
		data,
		nil,
	)
}

func (e Encrypter) Decrypt(ctx context.Context, data []byte) ([]byte, error) {
	_, span := system.NewSpan(ctx, system.GetTracer(), "pkg/verifier.Encrypter.Decrypt")
	defer span.End()

	privateKeyBytes, err := e.privateKey.Raw()
	if err != nil {
		return nil, err
	}
	rsaPrivateKey, err := x509.ParsePKCS1PrivateKey(privateKeyBytes)
	if err != nil {
		return nil, err
	}
	return rsa.DecryptOAEP(
		sha512.New(),
		rand.Reader,
		rsaPrivateKey,
		data,
		nil,
	)
}
