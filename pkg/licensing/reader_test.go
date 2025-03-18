//go:build unit || !integration

package licensing

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// cSpell:disable
const validOfficialTestLicense = "eyJhbGciOiJSUzI1NiIsImtpZCI6IjVuSm5GQ05TeUFUMVNRdnR6bDc4MllDZUdrV3FUQ3R2MWZ5SFVRa3hyTlUiLCJ0eXAiOiJKV1QifQ.eyJwcm9kdWN0IjoiQmFjYWxoYXUiLCJsaWNlbnNlX3ZlcnNpb24iOiJ2MSIsImxpY2Vuc2VfdHlwZSI6InN0YW5kYXJkIiwibGljZW5zZV9pZCI6ImU2NmQxZjNhLWE4ZDgtNGQ1Ny04ZjE0LTAwNzIyODQ0YWZlMiIsImN1c3RvbWVyX25hbWUiOiJiYWNhbGhhdS1pbnRlZ3JhdGlvbi10ZXN0cyIsImN1c3RvbWVyX2lkIjoidGVzdC1jdXN0b21lci1pZC0xMjMiLCJjYXBhYmlsaXRpZXMiOnsibWF4X25vZGVzIjoiMSJ9LCJtZXRhZGF0YSI6e30sImlhdCI6MTczNjg4MTYzOCwiaXNzIjoiaHR0cHM6Ly9leHBhbnNvLmlvLyIsInN1YiI6InRlc3QtY3VzdG9tZXItaWQtMTIzIiwiZXhwIjoyMzg0ODgxNjM4LCJqdGkiOiJlNjZkMWYzYS1hOGQ4LTRkNTctOGYxNC0wMDcyMjg0NGFmZTIifQ.U6qkWmki2wp3RbPdn8d0zzsy4FchZIyUDmJi2bJ4w4vhwJlJ0_F2_317v4iPzy9q69eJOKNaqj8P3xYaPbpiooFm15OdJ3ecbMy8bKvvWVj43stw6HNP_uoW-RlZnY2zTOQ9WhlOhjnUPPC-UXOcaMwxiLBwMo5n3Rs0W9uAQHGQIptGg0sKiZvIrMZZ3vww2PZ3wJDiDvznE2lPtI7jAbcFFKDlhY3UiXed2ihGTWvLW8Zwj4veCR4PAUoEDu-nfQDvlqNeAvABT-KrKY2M-d5T_WzK1WwXtHok9tG2OV5ybSZoxFDQW3iqiCg6TqMwCAa6C6MBXtLnv-NP1H9Ytg"

type ReaderTestSuite struct {
	suite.Suite
	tmpDir      string
	licensePath string
}

func (suite *ReaderTestSuite) SetupTest() {
	suite.tmpDir = suite.T().TempDir()
	suite.licensePath = filepath.Join(suite.tmpDir, "license.json")
}

func (suite *ReaderTestSuite) TestValidateLicense_InvalidToken() {
	licenseContent := `{"license": "invalid-token"}`
	err := os.WriteFile(suite.licensePath, []byte(licenseContent), 0644)
	suite.Require().NoError(err)

	manager, err := NewReader(suite.licensePath)
	suite.Require().Error(err)
	suite.Require().Nil(manager)
	suite.Require().ErrorContains(err, "license validation error: token is malformed: token contains an invalid number of segments")
}

func (suite *ReaderTestSuite) TestValidateLicense_ValidToken() {
	licenseContent := fmt.Sprintf(`{"license": %q}`, validOfficialTestLicense)
	err := os.WriteFile(suite.licensePath, []byte(licenseContent), 0644)
	suite.Require().NoError(err)

	manager, err := NewReader(suite.licensePath)
	suite.Require().NoError(err)
	suite.Require().NotNil(manager)

	claims := manager.License()
	suite.Require().NotNil(claims)

	// Verify basic claims
	assert.Equal(suite.T(), "Bacalhau", claims.Product)
	assert.Equal(suite.T(), "e66d1f3a-a8d8-4d57-8f14-00722844afe2", claims.LicenseID)
	assert.Equal(suite.T(), "standard", claims.LicenseType)
	assert.Equal(suite.T(), "test-customer-id-123", claims.CustomerID)
	assert.Equal(suite.T(), "v1", claims.LicenseVersion)
	assert.Equal(suite.T(), "1", claims.Capabilities["max_nodes"])
}

func (suite *ReaderTestSuite) TestValidateLicense_NoLicenseConfigured() {
	manager, err := NewReader("")
	suite.Require().NoError(err)
	suite.Require().NotNil(manager)

	claims := manager.License()
	suite.Require().Nil(claims)
}

func (suite *ReaderTestSuite) TestNewLicenseManager_InvalidJSON() {
	err := os.WriteFile(suite.licensePath, []byte("invalid json content"), 0644)
	suite.Require().NoError(err)

	manager, err := NewReader(suite.licensePath)
	suite.Require().Error(err)
	suite.Require().Nil(manager)
	suite.Require().ErrorContains(err, "failed to parse license file")
}

func (suite *ReaderTestSuite) TestNewLicenseManager_FileNotFound() {
	manager, err := NewReader("/non/existent/path/license.json")
	suite.Require().Error(err)
	suite.Require().Nil(manager)
	suite.Require().ErrorContains(err, "failed to read license file")
}

func (suite *ReaderTestSuite) TestNewLicenseManager_InvalidJSONStructure() {
	licenseContent := `{"some_other_field": "value"}`
	err := os.WriteFile(suite.licensePath, []byte(licenseContent), 0644)
	suite.Require().NoError(err)

	manager, err := NewReader(suite.licensePath)
	suite.Require().Error(err)
	suite.Require().Nil(manager)
	suite.Require().ErrorContains(err, "license validation error: token is malformed: token contains an invalid number of segments")
}

func TestReaderSuite(t *testing.T) {
	suite.Run(t, new(ReaderTestSuite))
}
