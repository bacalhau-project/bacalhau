package util

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"time"
)

const (
	permissions           = 0600
	rsaKeySize            = 4096
	serialNumberLimitBits = 128
)

var (
	ipv4Loopback = net.IPv4(127, 0, 0, 1) //nolint:gomnd
	ipv6Loopback = net.IPv6loopback
)

type Certificate struct {
	certFile     string
	keyFile      string
	certTemplate *x509.Certificate
	certPrivKey  *rsa.PrivateKey
}

type CACertificate struct {
	Certificate
}

func NewTestCACertificate(caCertPath, caKeyPath string) (*CACertificate, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), serialNumberLimitBits)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}

	ca := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Company, INC."},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, 1),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caPrivKey, err := rsa.GenerateKey(rand.Reader, rsaKeySize)
	if err != nil {
		return nil, err
	}
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, err
	}
	caPEM := new(bytes.Buffer)
	err = pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(caCertPath, caPEM.Bytes(), permissions)
	if err != nil {
		return nil, err
	}
	caPrivateKeyPEM := new(bytes.Buffer)
	err = pem.Encode(caPrivateKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	})
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(caKeyPath, caPrivateKeyPEM.Bytes(), permissions)
	if err != nil {
		return nil, err
	}
	return &CACertificate{
		Certificate: Certificate{
			certFile:     caCertPath,
			keyFile:      caKeyPath,
			certTemplate: ca,
			certPrivKey:  caPrivKey,
		},
	}, nil
}

func (c *CACertificate) CreateTestSignedCertificate(certPath, keyPath string) (*Certificate, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), serialNumberLimitBits)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}
	cert := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Test Server"},
		},
		IPAddresses: []net.IP{ipv4Loopback, ipv6Loopback},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(0, 0, 1),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,
	}

	certPrivKey, err := rsa.GenerateKey(rand.Reader, rsaKeySize)
	if err != nil {
		return nil, err
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, c.certTemplate, &certPrivKey.PublicKey, c.certPrivKey)
	if err != nil {
		return nil, err
	}
	certPEM := new(bytes.Buffer)
	err = pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(certPath, certPEM.Bytes(), permissions)
	if err != nil {
		return nil, err
	}

	certPrivKeyPEM := new(bytes.Buffer)
	err = pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	})
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(keyPath, certPrivKeyPEM.Bytes(), permissions)
	if err != nil {
		return nil, err
	}
	return &Certificate{
		certFile:     certPath,
		keyFile:      keyPath,
		certTemplate: cert,
	}, nil
}
