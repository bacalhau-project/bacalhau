package sso

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cSpell:disable
// setupTestCmd creates a new command and buffer for testing
func setupTokenTestCmd(t *testing.T) (*cobra.Command, *bytes.Buffer) {
	cmd := &cobra.Command{Use: "test"}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	return cmd, buf
}

// createTestTokenFile creates a temporary JWT token file for testing
func createTestTokenFile(t *testing.T, token string) (string, func()) {
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "jwt-tokens.json")
	// Write initial token data
	err := os.WriteFile(tokenPath, []byte(token), 0600)
	require.NoError(t, err)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}
	return tokenPath, cleanup
}

func TestToken_ShowRawToken(t *testing.T) {
	// Setup
	cmd, buf := setupTokenTestCmd(t)
	tokenPath, cleanup := createTestTokenFile(t, `{"http://test-api:1234":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"}`)
	defer cleanup()

	cfg := types.Bacalhau{
		API: types.API{
			Host: "test-api",
			Port: 1234,
		},
		DataDir: filepath.Dir(tokenPath),
	}

	o := &SSOTokenOptions{Decode: false}
	err := o.runSSOToken(cmd, cfg)

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9")
}

func TestToken_DecodeValidToken(t *testing.T) {
	// Setup
	cmd, buf := setupTokenTestCmd(t)
	// Using a valid JWT token
	tokenPath, cleanup := createTestTokenFile(t, `{"http://test-api:1234":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"}`)
	defer cleanup()

	cfg := types.Bacalhau{
		API: types.API{
			Host: "test-api",
			Port: 1234,
		},
		DataDir: filepath.Dir(tokenPath),
	}

	o := &SSOTokenOptions{Decode: true}
	err := o.runSSOToken(cmd, cfg)

	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Header:")
	assert.Contains(t, output, `"alg": "HS256"`)
	assert.Contains(t, output, "Claims:")
	assert.Contains(t, output, `"name": "John Doe"`)
}

func TestToken_NoTokenFound(t *testing.T) {
	// Setup
	cmd, buf := setupTokenTestCmd(t)
	tokenPath, cleanup := createTestTokenFile(t, `{}`)
	defer cleanup()

	cfg := types.Bacalhau{
		API: types.API{
			Host: "test-api",
			Port: 1234,
		},
		DataDir: filepath.Dir(tokenPath),
	}

	o := &SSOTokenOptions{Decode: false}
	err := o.runSSOToken(cmd, cfg)

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "No authentication token found")
}

func TestToken_DecodeInvalidToken(t *testing.T) {
	// Setup
	cmd, _ := setupTokenTestCmd(t)
	// Using an invalid JWT token
	tokenPath, cleanup := createTestTokenFile(t, `{"http://test-api:1234":"invalid-token"}`)
	defer cleanup()

	cfg := types.Bacalhau{
		API: types.API{
			Host: "test-api",
			Port: 1234,
		},
		DataDir: filepath.Dir(tokenPath),
	}

	o := &SSOTokenOptions{Decode: true}
	err := o.runSSOToken(cmd, cfg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse token")
}

func TestNewSSOTokenCmd(t *testing.T) {
	cmd := NewSSOTokenCmd()

	// Test command structure
	assert.Equal(t, "token", cmd.Use)
	assert.Equal(t, "Show current environment SSO JWT token", cmd.Short)

	// Test decode flag
	decodeFlag := cmd.Flags().Lookup("decode")
	assert.NotNil(t, decodeFlag)
	assert.Equal(t, "d", decodeFlag.Shorthand)
	assert.Equal(t, "false", decodeFlag.DefValue)
}
