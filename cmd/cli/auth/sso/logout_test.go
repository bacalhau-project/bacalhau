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

// setupTestCmd creates a new command and buffer for testing
func setupTestCmd(t *testing.T) (*cobra.Command, *bytes.Buffer) {
	cmd := &cobra.Command{Use: "test"}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	return cmd, buf
}

// createTempTokenFile creates a temporary JWT token file for testing
func createTempTokenFile(t *testing.T) (string, func()) {
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "jwt-tokens.json")
	// Write initial token data with exact format
	err := os.WriteFile(tokenPath, []byte(`{"http://test-api:1234":"test-token"}`), 0600)
	require.NoError(t, err)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}
	return tokenPath, cleanup
}

func TestLogout_ForceFlag(t *testing.T) {
	// Setup
	cmd, buf := setupTestCmd(t)
	tokenPath, cleanup := createTempTokenFile(t)
	defer cleanup()

	// Create test config with token path
	cfg := types.Bacalhau{
		API: types.API{
			Host: "test-api",
			Port: 1234,
		},
		DataDir: filepath.Dir(tokenPath), // Set DataDir directly in config
	}

	// Test with force flag
	o := &LogoutOptions{Force: true}
	err := o.runSSOLogout(cmd, cfg)

	// Assertions
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Successfully logged out from http://test-api:1234")

	// Verify token file is updated correctly
	content, err := os.ReadFile(tokenPath)
	require.NoError(t, err)
	expectedContent := "{}\n"
	assert.Equal(t, expectedContent, string(content))
}

func TestLogout_ConfirmYes(t *testing.T) {
	// Setup
	cmd, _ := setupTestCmd(t)
	tokenPath, cleanup := createTempTokenFile(t)
	defer cleanup()

	cfg := types.Bacalhau{
		API: types.API{
			Host: "test-api",
			Port: 1234,
		},
		DataDir: filepath.Dir(tokenPath),
	}

	// Create a buffer for input simulation with "y" response
	inBuf := bytes.NewBufferString("y\n")
	outBuf := new(bytes.Buffer)

	// Set up the command's input and output
	cmd.SetIn(inBuf)
	cmd.SetOut(outBuf)

	// Create a new reader for Scanf
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// Create a pipe and use it for stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	// Write "y" to the pipe
	go func() {
		w.Write([]byte("y\n"))
		w.Close()
	}()

	o := &LogoutOptions{Force: false}
	err := o.runSSOLogout(cmd, cfg)

	// Assertions
	assert.NoError(t, err)
	output := outBuf.String()
	assert.Contains(t, output, "Are you sure you want to logout")
	assert.Contains(t, output, "Successfully logged out")

	// Verify token file is updated
	content, err := os.ReadFile(tokenPath)
	require.NoError(t, err)
	expectedContent := "{}\n"
	assert.Equal(t, expectedContent, string(content))
}

func TestLogout_ConfirmNo(t *testing.T) {
	// Setup
	cmd, buf := setupTestCmd(t)
	tokenPath, cleanup := createTempTokenFile(t)
	defer cleanup()

	cfg := types.Bacalhau{
		API: types.API{
			Host: "test-api",
			Port: 1234,
		},
		DataDir: filepath.Dir(tokenPath), // Set DataDir directly in config
	}

	// Simulate user input "n"
	cmd.SetIn(bytes.NewBufferString("n\n"))

	o := &LogoutOptions{Force: false}
	err := o.runSSOLogout(cmd, cfg)

	// Assertions
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Logout cancelled")

	// Verify token file is unchanged
	content, err := os.ReadFile(tokenPath)
	require.NoError(t, err)
	expectedContent := `{"http://test-api:1234":"test-token"}`
	assert.Equal(t, expectedContent, string(content))
}

func TestNewSSOLogoutCmd(t *testing.T) {
	cmd := NewSSOLogoutCmd()

	// Test command structure
	assert.Equal(t, "logout", cmd.Use)
	assert.Equal(t, "Logout from current SSO session", cmd.Short)

	// Test force flag
	forceFlag := cmd.Flags().Lookup("force")
	assert.NotNil(t, forceFlag)
	assert.Equal(t, "f", forceFlag.Shorthand)
	assert.Equal(t, "false", forceFlag.DefValue)
}
