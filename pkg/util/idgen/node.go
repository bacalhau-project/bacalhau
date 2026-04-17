package idgen

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
)

// hostnameStringReplacer replaces dots with dashes in the hostname to make the node name valid for NATS subjects.
var hostnameStringReplacer = strings.NewReplacer(".", "-")

// NodeNameProvider defines an interface for generating node names.
type NodeNameProvider interface {
	GenerateNodeName(ctx context.Context) (string, error)
}

// NodeNameProviderFunc type is an adapter to allow the use of ordinary functions as NodeNameProvider.
type NodeNameProviderFunc func(ctx context.Context) (string, error)

// GenerateNodeName allows NodeNameProviderFunc to implement NodeNameProvider.
func (f NodeNameProviderFunc) GenerateNodeName(ctx context.Context) (string, error) {
	return f(ctx)
}

// HostnameProvider retrieves the node name from the host's hostname.
type HostnameProvider struct{}

func (HostnameProvider) GenerateNodeName(_ context.Context) (string, error) {
	h, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("retrieving hostname: %w", err)
	}
	return hostnameStringReplacer.Replace(h), nil
}

// AWSNodeNameProvider retrieves the node name from AWS instance metadata.
type AWSNodeNameProvider struct {
	httpProvider *HTTPNodeNameProvider
}

func NewAWSNodeNameProvider() AWSNodeNameProvider {
	return AWSNodeNameProvider{
		httpProvider: &HTTPNodeNameProvider{
			URL: "http://169.254.169.254/latest/meta-data/instance-id",
		},
	}
}

func (p AWSNodeNameProvider) GenerateNodeName(ctx context.Context) (string, error) {
	return p.httpProvider.GenerateNodeName(ctx)
}

// GCPNodeNameProvider retrieves the node name from GCP instance metadata.
type GCPNodeNameProvider struct {
	httpProvider *HTTPNodeNameProvider
}

func NewGCPNodeNameProvider() GCPNodeNameProvider {
	return GCPNodeNameProvider{
		httpProvider: &HTTPNodeNameProvider{
			URL:    "http://metadata.google.internal/computeMetadata/v1/instance/id",
			Header: map[string]string{"Metadata-Flavor": "Google"},
		},
	}
}

func (p GCPNodeNameProvider) GenerateNodeName(ctx context.Context) (string, error) {
	return p.httpProvider.GenerateNodeName(ctx)
}

// HTTPNodeNameProvider retrieves the node name from a URL, used by AWS and GCP.
type HTTPNodeNameProvider struct {
	URL    string
	Header map[string]string
}

func (h HTTPNodeNameProvider) GenerateNodeName(ctx context.Context) (string, error) {
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", h.URL, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	for key, value := range h.Header {
		req.Header.Add(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("performing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response body: %w", err)
	}
	return string(body), nil
}

// UUIDNodeNameProvider generates a random UUID as the node name.
type UUIDNodeNameProvider struct{}

func (UUIDNodeNameProvider) GenerateNodeName(_ context.Context) (string, error) {
	r, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("generating UUID: %w", err)
	}
	return r.String(), nil
}

// PUUIDNodeNameProvider generates a random UUID as the node name, with "n-" prefix.
type PUUIDNodeNameProvider struct{}

func (PUUIDNodeNameProvider) GenerateNodeName(ctx context.Context) (string, error) {
	res, err := UUIDNodeNameProvider{}.GenerateNodeName(ctx)
	if err != nil {
		return "", err
	}
	return NodeIDPrefix + res, nil
}

// CachedNodeNameProvider caches the node name for subsequent calls.
type CachedNodeNameProvider struct {
	provider NodeNameProvider
	name     string
}

func NewCachedNodeNameProvider(provider NodeNameProvider) *CachedNodeNameProvider {
	return &CachedNodeNameProvider{provider: provider}
}

func (c *CachedNodeNameProvider) GenerateNodeName(ctx context.Context) (string, error) {
	if c.name == "" {
		var err error
		c.name, err = c.provider.GenerateNodeName(ctx)
		if err != nil {
			return "", fmt.Errorf("caching node name: %w", err)
		}
	}
	return c.name, nil
}

// compile time check for NodeNameProvider interface
var _ NodeNameProvider = NodeNameProviderFunc(nil)
var _ NodeNameProvider = HostnameProvider{}
var _ NodeNameProvider = AWSNodeNameProvider{}
var _ NodeNameProvider = GCPNodeNameProvider{}
var _ NodeNameProvider = HTTPNodeNameProvider{}
var _ NodeNameProvider = UUIDNodeNameProvider{}
var _ NodeNameProvider = &CachedNodeNameProvider{}
