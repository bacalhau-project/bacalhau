//go:build unit || !integration

package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func getTestSelfSignedCert(t *testing.T) Certificate {
	key, err := rsa.GenerateKey(rand.Reader, rsaKeySize)
	require.NoError(t, err)

	cert, err := NewSelfSignedCertificate(key, true, nil)
	require.NoError(t, err)
	require.NotNil(t, cert)
	return cert
}

func TestProducesValidCertificate(t *testing.T) {
	cert := getTestSelfSignedCert(t)

	var buf bytes.Buffer
	err := cert.MarshalCertificate(&buf)
	require.NoError(t, err)

	block, rest := pem.Decode(buf.Bytes())
	require.NotNil(t, block)
	require.Empty(t, rest)

	parsed, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	require.True(t, parsed.IsCA)
}

func TestProducesSignedCertificate(t *testing.T) {
	parent := getTestSelfSignedCert(t)

	cert, err := NewSignedCertificate(parent, []net.IP{net.ParseIP("0.0.0.0")})
	require.NoError(t, err)
	require.NotNil(t, cert)

	var buf bytes.Buffer
	err = cert.MarshalCertificate(&buf)
	require.NoError(t, err)

	block, rest := pem.Decode(buf.Bytes())
	require.NotNil(t, block)
	require.Empty(t, rest)

	parsed, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)
	require.NotNil(t, parsed)

	buf.Reset()
	err = parent.MarshalCertificate(&buf)
	require.NoError(t, err)

	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(buf.Bytes())
	require.True(t, ok)

	chains, err := parsed.Verify(x509.VerifyOptions{
		Roots:     pool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	})
	require.NoError(t, err)
	require.NotEmpty(t, chains)
}
