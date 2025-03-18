package auth

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/sso"
	"github.com/stretchr/testify/assert"
)

// TestNewSSOCmd tests the creation of the SSO command
func TestNewSSOCmd(t *testing.T) {
	cmd := NewSSOCmd()

	assert.NotNil(t, cmd, "Command should not be nil")
	assert.Equal(t, "sso", cmd.Use, "Command use should be 'sso'")
	assert.Contains(t, cmd.Short, "Login using SSO", "Command should have appropriate short description")
}

// TestPrintDeviceCodeInstructions tests the output formatting of device code instructions
func TestPrintDeviceCodeInstructions(t *testing.T) {
	// Test case 1: With verificationURIComplete
	t.Run("With verification URI complete", func(t *testing.T) {
		// Redirect stdout for test
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		defer func() {
			os.Stdout = oldStdout
		}()

		deviceCode := &sso.DeviceCodeResponse{
			DeviceCode:              "device_code_123",
			UserCode:                "USER123",
			VerificationURI:         "https://example.com/verify",
			VerificationURIComplete: "https://example.com/verify?code=USER123",
			ExpiresIn:               300,
			Interval:                5,
		}
		providerName := "TestProvider"

		printDeviceCodeInstructions(deviceCode, providerName, w)

		w.Close()
		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		// Check that output contains all expected elements
		assert.Contains(t, output, "https://example.com/verify", "Output should contain verification URI")
		assert.Contains(t, output, "USER123", "Output should contain user code")
		assert.Contains(t, output, "https://example.com/verify?code=USER123", "Output should contain complete verification URI")
		assert.Contains(t, output, "TestProvider", "Output should contain provider name")
	})

	// Test case 2: Without verificationURIComplete
	t.Run("Without verification URI complete", func(t *testing.T) {
		// Redirect stdout for test
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		defer func() {
			os.Stdout = oldStdout
		}()

		deviceCode := &sso.DeviceCodeResponse{
			DeviceCode:      "device_code_123",
			UserCode:        "USER123",
			VerificationURI: "https://example.com/verify",
			ExpiresIn:       300,
			Interval:        5,
		}
		providerName := "AnotherProvider"

		printDeviceCodeInstructions(deviceCode, providerName, w)

		w.Close()
		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		// Check that output contains all expected elements but not the complete URI
		assert.Contains(t, output, "https://example.com/verify", "Output should contain verification URI")
		assert.Contains(t, output, "USER123", "Output should contain user code")
		assert.Contains(t, output, "AnotherProvider", "Output should contain provider name")
		assert.NotContains(t, output, "Or, open this URL in your browser", "Output should not mention alternative URL")
	})
}
