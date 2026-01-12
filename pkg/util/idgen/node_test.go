//go:build unit || !integration

package idgen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestHostnameProvider(t *testing.T) {
	provider := HostnameProvider{}
	name, err := provider.GenerateNodeName(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, name)
	hostname, _ := os.Hostname()
	assert.Equal(t, hostnameStringReplacer.Replace(hostname), name)
}

func TestAWSNodeNameProvider(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("aws-instance-id"))
	}))
	defer mockServer.Close() // httptest.Server.Close() doesn't return error

	awsProvider := NewAWSNodeNameProvider()
	awsProvider.httpProvider.URL = mockServer.URL // Replace the URL with the mock server's URL

	name, err := awsProvider.GenerateNodeName(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "aws-instance-id", name)
}

func TestGCPNodeNameProvider(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("gcp-instance-name"))
	}))
	defer mockServer.Close() // httptest.Server.Close() doesn't return error

	gcpProvider := NewGCPNodeNameProvider()
	gcpProvider.httpProvider.URL = mockServer.URL // Replace the URL with the mock server's URL

	name, err := gcpProvider.GenerateNodeName(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "gcp-instance-name", name)
}

func TestUUIDNodeNameProvider(t *testing.T) {
	provider := UUIDNodeNameProvider{}
	name, err := provider.GenerateNodeName(context.Background())
	assert.NoError(t, err)
	_, err = uuid.Parse(name)
	assert.NoError(t, err)
}

func TestPUUIDNodeNameProvider(t *testing.T) {
	provider := PUUIDNodeNameProvider{}
	name, err := provider.GenerateNodeName(context.Background())
	assert.NoError(t, err)
	assert.Truef(t, strings.HasPrefix(name, NodeIDPrefix), "expected %s to start with %s", name, NodeIDPrefix)
	_, err = uuid.Parse(name[2:])
	assert.NoError(t, err)
}

func TestCachedNodeNameProvider(t *testing.T) {
	mockProvider := NodeNameProviderFunc(func(ctx context.Context) (string, error) {
		return "cached-name", nil
	})
	cachedProvider := NewCachedNodeNameProvider(mockProvider)

	// First call should cache the name
	firstName, err := cachedProvider.GenerateNodeName(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "cached-name", firstName)

	// Modify the mock to return a different name
	mockProvider = func(ctx context.Context) (string, error) {
		return "new-name", nil
	}

	// Second call should return the cached name, not the new one
	secondName, err := cachedProvider.GenerateNodeName(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "cached-name", secondName)
}
