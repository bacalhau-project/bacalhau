//go:build unit || !integration

package license

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"os"
	"path/filepath"

	"encoding/json"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// cSpell:disable
const validOfficialTestLicense = "eyJhbGciOiJSUzI1NiIsImtpZCI6IjVuSm5GQ05TeUFUMVNRdnR6bDc4MllDZUdrV3FUQ3R2MWZ5SFVRa3hyTlUiLCJ0eXAiOiJKV1QifQ.eyJwcm9kdWN0IjoiQmFjYWxoYXUiLCJsaWNlbnNlX3ZlcnNpb24iOiJ2MSIsImxpY2Vuc2VfdHlwZSI6InN0YW5kYXJkIiwibGljZW5zZV9pZCI6ImU2NmQxZjNhLWE4ZDgtNGQ1Ny04ZjE0LTAwNzIyODQ0YWZlMiIsImN1c3RvbWVyX25hbWUiOiJiYWNhbGhhdS1pbnRlZ3JhdGlvbi10ZXN0cyIsImN1c3RvbWVyX2lkIjoidGVzdC1jdXN0b21lci1pZC0xMjMiLCJjYXBhYmlsaXRpZXMiOnsibWF4X25vZGVzIjoiMSJ9LCJtZXRhZGF0YSI6e30sImlhdCI6MTczNjg4MTYzOCwiaXNzIjoiaHR0cHM6Ly9leHBhbnNvLmlvLyIsInN1YiI6InRlc3QtY3VzdG9tZXItaWQtMTIzIiwiZXhwIjoyMzg0ODgxNjM4LCJqdGkiOiJlNjZkMWYzYS1hOGQ4LTRkNTctOGYxNC0wMDcyMjg0NGFmZTIifQ.U6qkWmki2wp3RbPdn8d0zzsy4FchZIyUDmJi2bJ4w4vhwJlJ0_F2_317v4iPzy9q69eJOKNaqj8P3xYaPbpiooFm15OdJ3ecbMy8bKvvWVj43stw6HNP_uoW-RlZnY2zTOQ9WhlOhjnUPPC-UXOcaMwxiLBwMo5n3Rs0W9uAQHGQIptGg0sKiZvIrMZZ3vww2PZ3wJDiDvznE2lPtI7jAbcFFKDlhY3UiXed2ihGTWvLW8Zwj4veCR4PAUoEDu-nfQDvlqNeAvABT-KrKY2M-d5T_WzK1WwXtHok9tG2OV5ybSZoxFDQW3iqiCg6TqMwCAa6C6MBXtLnv-NP1H9Ytg"

const validOfficialTestLicenseWithMetadata = "eyJhbGciOiJSUzI1NiIsImtpZCI6IjVuSm5GQ05TeUFUMVNRdnR6bDc4MllDZUdrV3FUQ3R2MWZ5SFVRa3hyTlUiLCJ0eXAiOiJKV1QifQ.eyJwcm9kdWN0IjoiQmFjYWxoYXUiLCJsaWNlbnNlX3ZlcnNpb24iOiJ2MSIsImxpY2Vuc2VfdHlwZSI6InN0YW5kYXJkIiwibGljZW5zZV9pZCI6IjJkNThjN2M5LWVjMjktNDVhNS1hNWNkLWNiOGY3ZmVlNjY3OCIsImN1c3RvbWVyX25hbWUiOiJiYWNhbGhhdS1pbnRlZ3JhdGlvbi10ZXN0cyIsImN1c3RvbWVyX2lkIjoidGVzdC1jdXN0b21lci1pZC0xMjMiLCJjYXBhYmlsaXRpZXMiOnsibWF4X25vZGVzIjoiMSJ9LCJtZXRhZGF0YSI6eyJzb21lTWV0YWRhdGEiOiJ2YWx1ZU9mU29tZU1ldGFkYXRhIn0sImlhdCI6MTczNjg4OTY4MiwiaXNzIjoiaHR0cHM6Ly9leHBhbnNvLmlvLyIsInN1YiI6InRlc3QtY3VzdG9tZXItaWQtMTIzIiwiZXhwIjoyMzg0ODg5NjgyLCJqdGkiOiIyZDU4YzdjOS1lYzI5LTQ1YTUtYTVjZC1jYjhmN2ZlZTY2NzgifQ.LDjEcSkGBHT6cHazgYYmviX6jxUPcEzVrkiyJ1QCgwdAswWusC2gWE-H7vu6X4rFFYV8hjycS2oJjaVLm4hLyGNvHPzRedIshGWM5j4GxoQ-p7ulf1HQErVMj5xzJzoyM0IwXN4Vb6h6AxNwYoey948Bduk--DeYBbMVwQAXyZeyb_A1jZeR3JLf1lQhoe6-cjmTnVMCNyzisZqHGYWpXHDYQcqSOm3FvPrBPsP4bVCZSU0pGQBu8lb9A3KhJRobvqNF4YseSz7fFkpuRR3sI7p4zthO6aEk7sXKF0LBU9G1AEdCn5S0gB-7_uFUuH_JQi8bhvXeWvC1dqdQLBzYnA"

const officialTestTokenWithWrongSignature = "eyJhbGciOiJSUzI1NiIsImtpZCI6IjVuSm5GQ05TeUFUMVNRdnR6bDc4MllDZUdrV3FUQ3R2MWZ5SFVRa3hyTlUiLCJ0eXAiOiJKV1QifQ.eyJwcm9kdWN0IjoiQmFjYWxoYXUiLCJsaWNlbnNlX3ZlcnNpb24iOiJ2MSIsImxpY2Vuc2VfdHlwZSI6InN0YW5kYXJkIiwibGljZW5zZV9pZCI6ImU2NmQxZjNhLWE4ZDgtNGQ1Ny04ZjE0LTAwNzIyODQ0YWZlMiIsImN1c3RvbWVyX25hbWUiOiJiYWNhbGhhdS1pbnRlZ3JhdGlvbi10ZXN0cyIsImN1c3RvbWVyX2lkIjoidGVzdC1jdXN0b21lci1pZC0xMjMiLCJjYXBhYmlsaXRpZXMiOnsibWF4X25vZGVzIjoiMSJ9LCJtZXRhZGF0YSI6e30sImlhdCI6MTczNjg4MTYzOCwiaXNzIjoiaHR0cHM6Ly9leHBhbnNvLmlvLyIsInN1YiI6InRlc3QtY3VzdG9tZXItaWQtMTIzIiwiZXhwIjoyMzg0ODgxNjM4LCJqdGkiOiJlNjZkMWYzYS1hOGQ4LTRkNTctOGYxNC0wMDcyMjg0NGFmZTIifQ.iambadsignature"

const officialTokenButExpired = "eyJhbGciOiJSUzI1NiIsImtpZCI6IjVuSm5GQ05TeUFUMVNRdnR6bDc4MllDZUdrV3FUQ3R2MWZ5SFVRa3hyTlUiLCJ0eXAiOiJKV1QifQ.eyJwcm9kdWN0IjoiQmFjYWxoYXUiLCJsaWNlbnNlX3ZlcnNpb24iOiJ2MSIsImxpY2Vuc2VfdHlwZSI6InN0YW5kYXJkIiwibGljZW5zZV9pZCI6IjBkZDA0Yzg0LTA5YjgtNDE3OS04OGY3LWM3MmE5ZDU2YzBhMiIsImN1c3RvbWVyX25hbWUiOiJiYWNhbGhhdS1pbnRlZ3JhdGlvbi10ZXN0cyIsImN1c3RvbWVyX2lkIjoidGVzdC1jdXN0b21lci1pZC0xMjMiLCJjYXBhYmlsaXRpZXMiOnsibWF4X25vZGVzIjoiMSJ9LCJtZXRhZGF0YSI6eyJzb21lTWV0YWRhdGEiOiJ2YWx1ZU9mU29tZU1ldGFkYXRhIn0sImlhdCI6MTczNjg5MTEzMSwiaXNzIjoiaHR0cHM6Ly9leHBhbnNvLmlvLyIsInN1YiI6InRlc3QtY3VzdG9tZXItaWQtMTIzIiwiZXhwIjoxNzM2MjQxMDk4LCJqdGkiOiIwZGQwNGM4NC0wOWI4LTQxNzktODhmNy1jNzJhOWQ1NmMwYTIifQ.URD1ofoJwrleEkXWQ7vWVv_gCzwM-1cR6_6SOIf-d7Uuh3ttFJdNMDw_gbZp65sgLMycQKkm5ngooxK-FSwVj6jl2c70SvzuEHbdUsSZClLReOSbmY7CO6bOQYzQYVEeoWiykVMdgj2REgnrP3b2n4KGyTFKoqqXYpdjSJ9BXXgw-RfkXmyBV1h8imymcXCZcYxzcKPSDSoZLUrPSqD5ooM021VKaTd4J4jFql3BrLGrvaRgUtSgfQdJjo1alMUalZ7hAEWkmhBlQ_ocdlHeJOR3Rrlk5c-JANOJ4UslMLG465QJ8tmfxaUbbOPB2YPj0f9uEbGW5kGkHW3BKQZbDQ"

func TestInspectMissingLicenseFlag(t *testing.T) {
	// Create a new command instance
	cmd := NewInspectCmd()

	// Set up buffer to capture output
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Test with no arguments
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s), received 0")

	// Test with empty value
	cmd.SetArgs([]string{""})
	err = cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "license file path cannot be empty")

	// Test with whitespace value
	cmd.SetArgs([]string{"   "})
	err = cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "license file path cannot be empty")
}

func TestInspectFileNotFound(t *testing.T) {
	// Create a new command instance
	cmd := NewInspectCmd()

	// Set up buffer to capture output
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Test with non-existent file
	cmd.SetArgs([]string{"non-existent-file.json"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file not found: non-existent-file.json")

	// Test with non-existent file in a non-existent directory
	cmd.SetArgs([]string{"non/existent/path/file.json"})
	err = cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file not found: non/existent/path/file.json")
}

func TestInspectCommandOutput(t *testing.T) {
	// Create a new command instance
	cmd := NewInspectCmd()

	// Create a temporary directory and file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test-license.json")

	// Create a valid license file
	licenseContent := fmt.Sprintf(`{
        "license": "%s"
    }`, validOfficialTestLicense)
	err := os.WriteFile(filePath, []byte(licenseContent), 0644)
	require.NoError(t, err, "Failed to create test license file")

	// Set up buffer to capture output
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Run command with the test file
	cmd.SetArgs([]string{filePath})
	err = cmd.Execute()
	require.NoError(t, err)

	// Check the output contains expected headers
	output := buf.String()

	expectedResult := `Product      = Bacalhau
License ID   = e66d1f3a-a8d8-4d57-8f14-00722844afe2
Customer ID  = test-customer-id-123
Valid Until  = 2045-07-28
Version      = v1
Expired      = false
Capabilities = max_nodes=1
Metadata     = {}`

	assert.Contains(t, output, expectedResult)
}

func TestInspectCommandYAMLOutput(t *testing.T) {
	// Create a new command instance
	cmd := NewInspectCmd()

	// Create a temporary directory and file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test-license.json")

	// Create a valid license file
	licenseContent := fmt.Sprintf(`{
        "license": "%s"
    }`, validOfficialTestLicense)
	err := os.WriteFile(filePath, []byte(licenseContent), 0644)
	require.NoError(t, err, "Failed to create test license file")

	// Set up buffer to capture output
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Run command with the test file and yaml output format
	cmd.SetArgs([]string{filePath, "--output", "yaml"})
	err = cmd.Execute()
	require.NoError(t, err)

	// Parse actual output
	var actualData map[string]interface{}
	err = yaml.Unmarshal(buf.Bytes(), &actualData)
	require.NoError(t, err, "Failed to parse actual YAML output")

	// Expected data
	expectedData := map[string]interface{}{
		"product":         "Bacalhau",
		"license_id":      "e66d1f3a-a8d8-4d57-8f14-00722844afe2",
		"license_type":    "standard",
		"customer_id":     "test-customer-id-123",
		"exp":             2384881638,
		"iat":             1736881638,
		"iss":             "https://expanso.io/",
		"jti":             "e66d1f3a-a8d8-4d57-8f14-00722844afe2",
		"sub":             "test-customer-id-123",
		"license_version": "v1",
		"capabilities":    map[string]interface{}{"max_nodes": "1"},
	}

	// Compare the maps
	assert.Equal(t, expectedData, actualData)
}

func TestInspectCommandJSONOutput(t *testing.T) {
	// Create a new command instance
	cmd := NewInspectCmd()

	// Create a temporary directory and file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test-license.json")

	// Create a valid license file
	licenseContent := fmt.Sprintf(`{
        "license": "%s"
    }`, validOfficialTestLicense)
	err := os.WriteFile(filePath, []byte(licenseContent), 0644)
	require.NoError(t, err, "Failed to create test license file")

	// Set up buffer to capture output
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Run command with the test file and json output format
	cmd.SetArgs([]string{filePath, "--output", "json"})
	err = cmd.Execute()
	require.NoError(t, err)

	// Parse actual output
	var actualData map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &actualData)
	require.NoError(t, err, "Failed to parse actual JSON output")

	// Convert the exp and iat values to int64 for consistent comparison
	if exp, ok := actualData["exp"].(float64); ok {
		actualData["exp"] = int64(exp)
	}
	if iat, ok := actualData["iat"].(float64); ok {
		actualData["iat"] = int64(iat)
	}

	// Expected data
	expectedData := map[string]interface{}{
		"product":         "Bacalhau",
		"license_id":      "e66d1f3a-a8d8-4d57-8f14-00722844afe2",
		"license_type":    "standard",
		"customer_id":     "test-customer-id-123",
		"exp":             int64(2384881638),
		"iat":             int64(1736881638),
		"iss":             "https://expanso.io/",
		"jti":             "e66d1f3a-a8d8-4d57-8f14-00722844afe2",
		"sub":             "test-customer-id-123",
		"license_version": "v1",
		"capabilities":    map[string]interface{}{"max_nodes": "1"},
	}

	// Compare the maps
	assert.Equal(t, expectedData, actualData)
}

func TestInspectValidLicenseFile(t *testing.T) {
	// Create a new command instance
	cmd := NewInspectCmd()

	// Create a temporary directory and file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test-license.json")

	// Create a valid license file with the JWT token
	licenseContent := fmt.Sprintf(`{
        "license": "%s"
    }`, validOfficialTestLicense)
	err := os.WriteFile(filePath, []byte(licenseContent), 0644)
	require.NoError(t, err, "Failed to create test license file")

	// Set up buffer to capture output
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Test key-value output
	cmd.SetArgs([]string{filePath})
	err = cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	expectedOutput := `Product      = Bacalhau
License ID   = e66d1f3a-a8d8-4d57-8f14-00722844afe2
Customer ID  = test-customer-id-123
Valid Until  = 2045-07-28
Version      = v1
Expired      = false
Capabilities = max_nodes=1
Metadata     = {}`

	assert.Equal(t, expectedOutput, strings.TrimSpace(output))

	// Test JSON output
	buf.Reset()
	cmd.SetArgs([]string{filePath, "--output", "json"})
	err = cmd.Execute()
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	// Convert timestamp fields to int64 for comparison
	if exp, ok := result["exp"].(float64); ok {
		result["exp"] = int64(exp)
	}
	if iat, ok := result["iat"].(float64); ok {
		result["iat"] = int64(iat)
	}

	expectedJSON := map[string]interface{}{
		"product":         "Bacalhau",
		"license_id":      "e66d1f3a-a8d8-4d57-8f14-00722844afe2",
		"license_type":    "standard",
		"customer_id":     "test-customer-id-123",
		"exp":             int64(2384881638),
		"iat":             int64(1736881638),
		"iss":             "https://expanso.io/",
		"jti":             "e66d1f3a-a8d8-4d57-8f14-00722844afe2",
		"sub":             "test-customer-id-123",
		"license_version": "v1",
		"capabilities":    map[string]interface{}{"max_nodes": "1"},
	}

	assert.Equal(t, expectedJSON, result)
}

func TestInspectValidLicenseFileWithMetadata(t *testing.T) {
	// Create a new command instance
	cmd := NewInspectCmd()

	// Create a temporary directory and file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test-license.json")

	// Create a valid license file with the JWT token
	licenseContent := fmt.Sprintf(`{
        "license": "%s"
    }`, validOfficialTestLicenseWithMetadata)
	err := os.WriteFile(filePath, []byte(licenseContent), 0644)
	require.NoError(t, err, "Failed to create test license file")

	// Set up buffer to capture output
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Test key-value output
	cmd.SetArgs([]string{filePath})
	err = cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	expectedOutput := `Product      = Bacalhau
License ID   = 2d58c7c9-ec29-45a5-a5cd-cb8f7fee6678
Customer ID  = test-customer-id-123
Valid Until  = 2045-07-28
Version      = v1
Expired      = false
Capabilities = max_nodes=1
Metadata     = someMetadata=valueOfSomeMetadata`

	assert.Equal(t, expectedOutput, strings.TrimSpace(output))

	// Test JSON output
	buf.Reset()
	cmd.SetArgs([]string{filePath, "--output", "json"})
	err = cmd.Execute()
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	// Convert timestamp fields to int64 for comparison
	if exp, ok := result["exp"].(float64); ok {
		result["exp"] = int64(exp)
	}
	if iat, ok := result["iat"].(float64); ok {
		result["iat"] = int64(iat)
	}

	expectedJSON := map[string]interface{}{
		"product":         "Bacalhau",
		"license_id":      "2d58c7c9-ec29-45a5-a5cd-cb8f7fee6678",
		"license_type":    "standard",
		"customer_id":     "test-customer-id-123",
		"exp":             int64(2384889682),
		"iat":             int64(1736889682),
		"iss":             "https://expanso.io/",
		"jti":             "2d58c7c9-ec29-45a5-a5cd-cb8f7fee6678",
		"sub":             "test-customer-id-123",
		"license_version": "v1",
		"capabilities":    map[string]interface{}{"max_nodes": "1"},
		"metadata":        map[string]interface{}{"someMetadata": "valueOfSomeMetadata"},
	}

	assert.Equal(t, expectedJSON, result)
}

func TestInspectInvalidLicenseToken(t *testing.T) {
	cmd := NewInspectCmd()
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "invalid-token.json")

	// Create a file with invalid token
	licenseContent := `{
        "license": "invalid.jwt.token"
    }`
	err := os.WriteFile(filePath, []byte(licenseContent), 0644)
	require.NoError(t, err, "Failed to create test file")

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{filePath})

	err = cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid license")
}

func TestInspectInvalidSignatureLicenseToken(t *testing.T) {
	cmd := NewInspectCmd()
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "invalid-signature.json")

	// Create a file with a token that has invalid signature
	licenseContent := fmt.Sprintf(`{
        "license": "%s"
    }`, officialTestTokenWithWrongSignature)
	err := os.WriteFile(filePath, []byte(licenseContent), 0644)
	require.NoError(t, err, "Failed to create test file")

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{filePath})

	err = cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid license: license validation error: token signature is invalid")
}

func TestInspectExpiredLicenseToken(t *testing.T) {
	cmd := NewInspectCmd()
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "expired-license.json")

	// Create a file with a token that has invalid signature
	licenseContent := fmt.Sprintf(`{
        "license": "%s"
    }`, officialTokenButExpired)
	err := os.WriteFile(filePath, []byte(licenseContent), 0644)
	require.NoError(t, err, "Failed to create test file")

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{filePath})

	err = cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	expectedOutput := `Product      = Bacalhau
License ID   = 0dd04c84-09b8-4179-88f7-c72a9d56c0a2
Customer ID  = test-customer-id-123
Valid Until  = 2025-01-07
Version      = v1
Expired      = true
Capabilities = max_nodes=1
Metadata     = someMetadata=valueOfSomeMetadata`

	assert.Equal(t, expectedOutput, strings.TrimSpace(output))
}

func TestInspectMalformedLicenseFile(t *testing.T) {
	cmd := NewInspectCmd()
	tmpDir := t.TempDir()

	testCases := []struct {
		name        string
		content     string
		expectedErr string
	}{
		{
			name:        "not json",
			content:     "this is not json",
			expectedErr: "failed to parse license file",
		},
		{
			name:        "missing license key",
			content:     `{"some_other_key": "value"}`,
			expectedErr: "invalid license: license validation error: token is malformed",
		},
		{
			name:        "random string as license",
			content:     `{"license": "some random string"}`,
			expectedErr: "invalid license: license validation error: token is malformed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filePath := filepath.Join(tmpDir, fmt.Sprintf("malformed-%s.json", tc.name))
			err := os.WriteFile(filePath, []byte(tc.content), 0644)
			require.NoError(t, err, "Failed to create test file")

			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs([]string{filePath})

			err = cmd.Execute()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedErr)
		})
	}
}
