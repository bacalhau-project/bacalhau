package auth

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/common"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/stretchr/testify/require"
)

// mockAPI implements client.API interface for testing
type mockAPI struct {
	nodeAuthConfig *apimodels.GetAgentNodeAuthConfigResponse
	err            error
}

func (m *mockAPI) Agent() *client.Agent {
	return &client.Agent{}
}

func (m *mockAPI) Auth() *client.Auth {
	return &client.Auth{}
}

func (m *mockAPI) Jobs() *client.Jobs {
	return &client.Jobs{}
}

func (m *mockAPI) Nodes() *client.Nodes {
	return &client.Nodes{}
}

// mockClient implements client.Client interface
type mockClient struct {
	nodeAuthConfig *apimodels.GetAgentNodeAuthConfigResponse
	err            error
}

func (m *mockClient) Get(ctx context.Context, path string, req apimodels.GetRequest, res apimodels.GetResponse) error {
	if path == "/api/v1/agent/authconfig" {
		if m.err != nil {
			return m.err
		}
		// Type assert res to *apimodels.GetAgentNodeAuthConfigResponse
		if authRes, ok := res.(*apimodels.GetAgentNodeAuthConfigResponse); ok {
			*authRes = *m.nodeAuthConfig
		}
	}
	return nil
}

func (m *mockClient) Post(ctx context.Context, path string, req apimodels.PutRequest, res apimodels.PutResponse) error {
	return nil
}

func (m *mockClient) Put(ctx context.Context, path string, req apimodels.PutRequest, res apimodels.PutResponse) error {
	return nil
}

func (m *mockClient) Delete(ctx context.Context, path string, req apimodels.PutRequest, res apimodels.Response) error {
	return nil
}

func (m *mockClient) List(ctx context.Context, path string, req apimodels.ListRequest, res apimodels.ListResponse) error {
	return nil
}

func (m *mockClient) Dial(ctx context.Context, path string, req apimodels.Request) (<-chan *concurrency.AsyncResult[[]byte], error) {
	return nil, nil
}

// TestInfo_NoSSOSupport tests when the server doesn't support any auth methods
func TestInfo_NoSSOSupport(t *testing.T) {
	// Create a mock client that returns an error
	mockClient := &mockClient{
		err: fmt.Errorf("auth not supported"),
	}

	api := client.NewAPI(mockClient)

	var out bytes.Buffer
	cmd := NewInfoCmd()
	cmd.SetOut(&out)
	o := NewInfoOptions()
	err := o.runInfo(cmd, api, types.Bacalhau{})

	require.NoError(t, err)

	output := out.String()
	// Test all environment variable sections
	require.Contains(t, output, "Environment Variables:")
	require.Contains(t, output, "API Key: Not Set")
	require.Contains(t, output, "Username: Not Set")
	require.Contains(t, output, "Password: Not Set")

	// Test server support message
	require.Contains(t, output, "Server does not support Basic Auth, API Keys, or SSO logins")

	// Test that SSO section is not present
	require.NotContains(t, output, "Node SSO Authentication:")
	require.NotContains(t, output, "Provider Name:")
	require.NotContains(t, output, "Provider ID:")
	require.NotContains(t, output, "Version:")
}

// TestInfo_WithSSOSupport tests when the server supports SSO
func TestInfo_WithSSOSupport(t *testing.T) {
	// Create a mock client that returns SSO config
	mockClient := &mockClient{
		nodeAuthConfig: &apimodels.GetAgentNodeAuthConfigResponse{
			Config: types.Oauth2Config{
				ProviderName: "github",
				ProviderID:   "github-provider",
			},
			Version: "v1",
		},
	}

	api := client.NewAPI(mockClient)

	var out bytes.Buffer
	cmd := NewInfoCmd()
	cmd.SetOut(&out)
	o := NewInfoOptions()
	err := o.runInfo(cmd, api, types.Bacalhau{})

	require.NoError(t, err)

	output := out.String()
	// Test environment variables section
	require.Contains(t, output, "Environment Variables:")
	require.Contains(t, output, "API Key: Not Set")
	require.Contains(t, output, "Username: Not Set")
	require.Contains(t, output, "Password: Not Set")

	// Test SSO configuration section
	require.Contains(t, output, "Node SSO Authentication:")
	require.Contains(t, output, "Provider Name: github")
	require.Contains(t, output, "Provider ID: github-provider")
	require.Contains(t, output, "Version: v1")

	// Test environment variable note
	require.Contains(t, output, "Note: Environment variables take precedence")
	require.Contains(t, output, "To use SSO login, please unset Auth related environment variables")
}

// TestInfo_NoSSOConfig tests when the server responds but has no SSO config
func TestInfo_NoSSOConfig(t *testing.T) {
	// Create a mock client that returns empty config
	mockClient := &mockClient{
		nodeAuthConfig: &apimodels.GetAgentNodeAuthConfigResponse{
			Config:  types.Oauth2Config{},
			Version: "v1",
		},
	}

	api := client.NewAPI(mockClient)

	var out bytes.Buffer
	cmd := NewInfoCmd()
	cmd.SetOut(&out)
	o := NewInfoOptions()
	err := o.runInfo(cmd, api, types.Bacalhau{})

	require.NoError(t, err)

	output := out.String()
	// Test environment variables section
	require.Contains(t, output, "Environment Variables:")
	require.Contains(t, output, "API Key: Not Set")
	require.Contains(t, output, "Username: Not Set")
	require.Contains(t, output, "Password: Not Set")

	// Test SSO section
	require.Contains(t, output, "Node SSO Authentication:")
	require.Contains(t, output, "Server does not support SSO login")

	// Test that SSO details are not present
	require.NotContains(t, output, "Provider Name:")
	require.NotContains(t, output, "Provider ID:")
	require.NotContains(t, output, "Version:")

	// Test environment variable note
	require.Contains(t, output, "Note: Environment variables take precedence")
	require.Contains(t, output, "To use SSO login, please unset Auth related environment variables")
}

// TestInfo_WithEnvironmentVariables tests when environment variables are set
func TestInfo_WithEnvironmentVariables(t *testing.T) {
	// Set environment variables
	os.Setenv(common.BacalhauApiKey, "test-key")
	os.Setenv(common.BacalhauApiUsername, "test-user")
	os.Setenv(common.BacalhauApiPassword, "test-pass")
	defer func() {
		os.Unsetenv(common.BacalhauApiKey)
		os.Unsetenv(common.BacalhauApiUsername)
		os.Unsetenv(common.BacalhauApiPassword)
	}()

	// Create a mock client
	mockClient := &mockClient{
		nodeAuthConfig: &apimodels.GetAgentNodeAuthConfigResponse{
			Config: types.Oauth2Config{
				ProviderName: "github",
				ProviderID:   "github-provider",
			},
			Version: "v1",
		},
	}

	api := client.NewAPI(mockClient)

	var out bytes.Buffer
	cmd := NewInfoCmd()
	cmd.SetOut(&out)
	o := NewInfoOptions()
	err := o.runInfo(cmd, api, types.Bacalhau{})

	require.NoError(t, err)

	output := out.String()
	// Test environment variables section
	require.Contains(t, output, "Environment Variables:")
	require.Contains(t, output, "API Key: Set")
	require.Contains(t, output, "Username: Set")
	require.Contains(t, output, "Password: Set")

	// Test SSO configuration section
	require.Contains(t, output, "Node SSO Authentication:")
	require.Contains(t, output, "Provider Name: github")
	require.Contains(t, output, "Provider ID: github-provider")
	require.Contains(t, output, "Version: v1")

	// Test environment variable note
	require.Contains(t, output, "Note: Environment variables take precedence")
	require.Contains(t, output, "To use SSO login, please unset Auth related environment variables")
}
