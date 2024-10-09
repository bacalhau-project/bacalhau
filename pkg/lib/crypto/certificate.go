package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"time"
)

const (
	rsaKeySize            = 4096
	serialNumberLimitBits = 128
)

type Certificate struct {
	cert   *x509.Certificate
	parent *Certificate
	key    *rsa.PrivateKey
}

func NewSelfSignedCertificate(key *rsa.PrivateKey, isCA bool, ipAddresses []net.IP) (Certificate, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), serialNumberLimitBits)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return Certificate{}, err
	}

	usage := x509.KeyUsageDigitalSignature
	if isCA {
		usage |= x509.KeyUsageCertSign
	}

	cert := &x509.Certificate{
		SerialNumber:          serialNumber,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		IPAddresses:           ipAddresses,
		IsCA:                  isCA,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              usage,
		BasicConstraintsValid: true,
	}

	return Certificate{cert: cert, parent: nil, key: key}, nil
}

func NewSignedCertificate(parent Certificate, ipAddress []net.IP) (Certificate, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), serialNumberLimitBits)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return Certificate{}, err
	}
	cert := &x509.Certificate{
		SerialNumber:          serialNumber,
		IPAddresses:           ipAddress,
		IsCA:                  false,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	certPrivKey, err := rsa.GenerateKey(rand.Reader, rsaKeySize)
	if err != nil {
		return Certificate{}, err
	}

	return Certificate{cert: cert, parent: &parent, key: certPrivKey}, nil
}

func (cert *Certificate) MarshalCertificate(out io.Writer) error {
	var parent *x509.Certificate
	var signingKey *rsa.PrivateKey

	if cert.parent != nil {
		parent = cert.parent.cert
		signingKey = cert.parent.key
	} else {
		parent = cert.cert
		signingKey = cert.key
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, cert.cert, parent, &cert.key.PublicKey, signingKey)
	if err != nil {
		return err
	}

	caPEM := new(bytes.Buffer)
	err = pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})
	if err != nil {
		return err
	}

	n, err := out.Write(caPEM.Bytes())
	if err != nil {
		return err
	} else if n != caPEM.Len() {
		return fmt.Errorf("failed to completely write certificate")
	}

	return nil
}

func (cert *Certificate) MarshalPrivateKey(out io.Writer) error {
	certPrivKeyPEM := new(bytes.Buffer)
	err := pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(cert.key),
	})
	if err != nil {
		return err
	}

	n, err := out.Write(certPrivKeyPEM.Bytes())
	if err != nil {
		return err
	} else if n != certPrivKeyPEM.Len() {
		return fmt.Errorf("failed to completely write certificate private key")
	}
	return nil
}
