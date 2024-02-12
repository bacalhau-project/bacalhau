package certificates

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

type Certificate struct {
	certFile     string
	keyFile      string
	certTemplate *x509.Certificate
	certPrivKey  *rsa.PrivateKey
}

type CACertificate struct {
	Certificate
}

func NewCACertificate(caCertPath, caKeyPath string) (*CACertificate, error) {
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
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
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, err
	}
	caPEM := new(bytes.Buffer)
	pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})
	err = os.WriteFile(caCertPath, caPEM.Bytes(), 0644)
	if err != nil {
		return nil, err
	}
	caPrivateKeyPEM := new(bytes.Buffer)
	pem.Encode(caPrivateKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	})
	err = os.WriteFile(caKeyPath, caPrivateKeyPEM.Bytes(), 0600)
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

func (c *CACertificate) CreateSignedCertificate(CertPath, KeyPath string) (*Certificate, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}
	cert := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Test Server"},
		},
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(0, 0, 1),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,
	}

	certPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}
	//Sign the certificate with the CA
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, c.certTemplate, &certPrivKey.PublicKey, c.certPrivKey)
	if err != nil {
		return nil, err
	}
	certPEM := new(bytes.Buffer)
	pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	err = os.WriteFile(CertPath, certPEM.Bytes(), 0644)
	if err != nil {
		return nil, err
	}

	certPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	})
	err = os.WriteFile(KeyPath, certPrivKeyPEM.Bytes(), 0600)
	if err != nil {
		return nil, err
	}
	return &Certificate{
		certFile:     CertPath,
		keyFile:      KeyPath,
		certTemplate: cert,
	}, nil
}
