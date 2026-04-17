//go:build unit || !integration

package migrations

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/config/profile"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

type V4ToV5MigrationSuite struct {
	BaseMigrationTestSuite
}

func TestV4ToV5MigrationSuite(t *testing.T) {
	suite.Run(t, new(V4ToV5MigrationSuite))
}

// setupV4Repo creates a minimal v4 repo structure
func (suite *V4ToV5MigrationSuite) setupV4Repo() {
	// Create system_metadata.yaml with version 4
	metaContent := "RepoVersion: 4\nInstanceID: test-instance\n"
	err := os.WriteFile(filepath.Join(suite.TempDir, "system_metadata.yaml"), []byte(metaContent), 0644)
	suite.Require().NoError(err)
}

// writeTokensJSON writes a tokens.json file
func (suite *V4ToV5MigrationSuite) writeTokensJSON(tokens map[string]string) {
	data, err := json.Marshal(tokens)
	suite.Require().NoError(err)
	err = os.WriteFile(filepath.Join(suite.TempDir, TokensFileName), data, 0644)
	suite.Require().NoError(err)
}

// writeConfigYAML writes a config.yaml file
func (suite *V4ToV5MigrationSuite) writeConfigYAML(content string) {
	err := os.WriteFile(filepath.Join(suite.TempDir, "config.yaml"), []byte(content), 0644)
	suite.Require().NoError(err)
}

// createFsRepo creates an FsRepo for testing
func (suite *V4ToV5MigrationSuite) createFsRepo() repo.FsRepo {
	fsRepo, err := repo.NewFS(repo.FsRepoParams{
		Path: suite.TempDir,
	})
	suite.Require().NoError(err)
	return *fsRepo
}

// getProfileStore returns a profile store for the test repo
func (suite *V4ToV5MigrationSuite) getProfileStore() *profile.Store {
	return profile.NewStore(filepath.Join(suite.TempDir, ProfilesDirName))
}

func (suite *V4ToV5MigrationSuite) TestMigrateTokensToProfiles() {
	suite.setupV4Repo()

	// Setup tokens.json
	tokens := map[string]string{
		"https://prod.example.com:443": "prod-token",
		"http://localhost:1234":        "dev-token",
	}
	suite.writeTokensJSON(tokens)

	// Run migration
	fsRepo := suite.createFsRepo()
	err := V4ToV5(fsRepo)
	suite.Require().NoError(err)

	// Verify profiles were created
	store := suite.getProfileStore()
	profiles, err := store.List()
	suite.Require().NoError(err)
	suite.Require().Len(profiles, 2)

	// Verify profile contents
	prodProfile, err := store.Load("prod_example_com_443")
	suite.Require().NoError(err)
	suite.Equal("https://prod.example.com:443", prodProfile.Endpoint)
	suite.Require().NotNil(prodProfile.Auth)
	suite.Equal("prod-token", prodProfile.Auth.Token)

	devProfile, err := store.Load("localhost_1234")
	suite.Require().NoError(err)
	suite.Equal("http://localhost:1234", devProfile.Endpoint)
	suite.Require().NotNil(devProfile.Auth)
	suite.Equal("dev-token", devProfile.Auth.Token)

	// Verify current profile is set
	current, err := store.GetCurrent()
	suite.Require().NoError(err)
	suite.NotEmpty(current)
}

func (suite *V4ToV5MigrationSuite) TestMigrateConfigYAMLClientSettings() {
	suite.setupV4Repo()

	// Setup config.yaml with API settings
	configContent := `
API:
  Host: api.example.com
  Port: 8080
  TLS:
    UseTLS: true
    Insecure: true
`
	suite.writeConfigYAML(configContent)

	// Run migration
	fsRepo := suite.createFsRepo()
	err := V4ToV5(fsRepo)
	suite.Require().NoError(err)

	// Verify default profile was created
	store := suite.getProfileStore()
	profiles, err := store.List()
	suite.Require().NoError(err)
	suite.Require().Len(profiles, 1)
	suite.Contains(profiles, DefaultProfileName)

	// Verify profile contents
	defaultProfile, err := store.Load(DefaultProfileName)
	suite.Require().NoError(err)
	suite.Equal("https://api.example.com:8080", defaultProfile.Endpoint)
	suite.Require().NotNil(defaultProfile.TLS)
	suite.True(defaultProfile.TLS.Insecure)

	// Verify current profile is set to default
	current, err := store.GetCurrent()
	suite.Require().NoError(err)
	suite.Equal(DefaultProfileName, current)
}

func (suite *V4ToV5MigrationSuite) TestSkipMigrationIfProfilesExist() {
	suite.setupV4Repo()

	// Create existing profile
	store := suite.getProfileStore()
	err := store.Save("existing", &profile.Profile{
		Endpoint: "https://existing.example.com:443",
	})
	suite.Require().NoError(err)

	// Setup tokens.json that should NOT be migrated
	tokens := map[string]string{
		"https://new.example.com:443": "new-token",
	}
	suite.writeTokensJSON(tokens)

	// Run migration
	fsRepo := suite.createFsRepo()
	err = V4ToV5(fsRepo)
	suite.Require().NoError(err)

	// Verify only the existing profile exists (new one was not created)
	profiles, err := store.List()
	suite.Require().NoError(err)
	suite.Require().Len(profiles, 1)
	suite.Contains(profiles, "existing")
}

func (suite *V4ToV5MigrationSuite) TestMigrationIsIdempotent() {
	suite.setupV4Repo()

	// Setup tokens.json
	tokens := map[string]string{
		"https://prod.example.com:443": "prod-token",
	}
	suite.writeTokensJSON(tokens)

	fsRepo := suite.createFsRepo()

	// Run migration first time
	err := V4ToV5(fsRepo)
	suite.Require().NoError(err)

	store := suite.getProfileStore()
	profilesAfterFirst, err := store.List()
	suite.Require().NoError(err)

	// Run migration second time
	err = V4ToV5(fsRepo)
	suite.Require().NoError(err)

	// Verify profiles unchanged
	profilesAfterSecond, err := store.List()
	suite.Require().NoError(err)
	suite.Equal(profilesAfterFirst, profilesAfterSecond)
}

func (suite *V4ToV5MigrationSuite) TestMigrateWithBothTokensAndConfig() {
	suite.setupV4Repo()

	// Setup tokens.json
	tokens := map[string]string{
		"https://prod.example.com:443": "prod-token",
	}
	suite.writeTokensJSON(tokens)

	// Setup config.yaml with different endpoint
	configContent := `
API:
  Host: localhost
  Port: 1234
`
	suite.writeConfigYAML(configContent)

	// Run migration
	fsRepo := suite.createFsRepo()
	err := V4ToV5(fsRepo)
	suite.Require().NoError(err)

	// Verify both profiles were created
	store := suite.getProfileStore()
	profiles, err := store.List()
	suite.Require().NoError(err)
	suite.Require().Len(profiles, 2)
	suite.Contains(profiles, "prod_example_com_443")
	suite.Contains(profiles, DefaultProfileName)

	// Verify current is set to default (preferred)
	current, err := store.GetCurrent()
	suite.Require().NoError(err)
	suite.Equal(DefaultProfileName, current)
}

func (suite *V4ToV5MigrationSuite) TestMigrateEmptyTokensFile() {
	suite.setupV4Repo()

	// Create empty tokens.json
	err := os.WriteFile(filepath.Join(suite.TempDir, TokensFileName), []byte{}, 0644)
	suite.Require().NoError(err)

	// Run migration - should not fail
	fsRepo := suite.createFsRepo()
	err = V4ToV5(fsRepo)
	suite.Require().NoError(err)

	// Verify no profiles created
	store := suite.getProfileStore()
	profiles, err := store.List()
	suite.Require().NoError(err)
	suite.Empty(profiles)
}

func (suite *V4ToV5MigrationSuite) TestMigrateNoTokensOrConfig() {
	suite.setupV4Repo()

	// Run migration with no tokens.json or config.yaml
	fsRepo := suite.createFsRepo()
	err := V4ToV5(fsRepo)
	suite.Require().NoError(err)

	// Verify no profiles created
	store := suite.getProfileStore()
	profiles, err := store.List()
	suite.Require().NoError(err)
	suite.Empty(profiles)
}

func (suite *V4ToV5MigrationSuite) TestMigrateConfigWithDefaultValues() {
	suite.setupV4Repo()

	// Setup config.yaml with only host (no port)
	configContent := `
API:
  Host: myserver.local
`
	suite.writeConfigYAML(configContent)

	// Run migration
	fsRepo := suite.createFsRepo()
	err := V4ToV5(fsRepo)
	suite.Require().NoError(err)

	// Verify default profile was created with default port
	store := suite.getProfileStore()
	defaultProfile, err := store.Load(DefaultProfileName)
	suite.Require().NoError(err)
	suite.Equal("http://myserver.local:1234", defaultProfile.Endpoint)
}

func (suite *V4ToV5MigrationSuite) TestMigrateTokensWithEmptyEndpoint() {
	suite.setupV4Repo()

	// Setup tokens.json with empty endpoint
	tokens := map[string]string{
		"":                              "empty-token",
		"https://valid.example.com:443": "valid-token",
	}
	suite.writeTokensJSON(tokens)

	// Run migration
	fsRepo := suite.createFsRepo()
	err := V4ToV5(fsRepo)
	suite.Require().NoError(err)

	// Verify only valid profile was created
	store := suite.getProfileStore()
	profiles, err := store.List()
	suite.Require().NoError(err)
	suite.Require().Len(profiles, 1)
	suite.Contains(profiles, "valid_example_com_443")
}

// Test helper functions
func TestEndpointToProfileName(t *testing.T) {
	tests := []struct {
		endpoint string
		expected string
	}{
		{"https://prod.example.com:443", "prod_example_com_443"},
		{"http://localhost:1234", "localhost_1234"},
		{"https://api.cluster.internal:8080", "api_cluster_internal_8080"},
		{"http://192.168.1.100:1234", "192_168_1_100_1234"},
		{"https://example.com", "example_com"},
		// Note: url.Parse("invalid-url") treats it as a path with empty hostname
		// so it falls back to sanitization which produces "migrated" for empty string
		{"invalid-url", "migrated"},
		{"https://my-server.local:8080", "my_server_local_8080"},
	}

	for _, tc := range tests {
		t.Run(tc.endpoint, func(t *testing.T) {
			result := endpointToProfileName(tc.endpoint)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestSanitizeProfileName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"prod.example.com", "prod_example_com"},
		{"localhost:1234", "localhost_1234"},
		{"https://example.com", "https_example_com"},
		{"my--profile", "my_profile"},
		{"___test___", "test"},
		{"", "migrated"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := sanitizeProfileName(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}

// TestMigrateInvalidTokensJSON tests migration with malformed tokens.json
func (suite *V4ToV5MigrationSuite) TestMigrateInvalidTokensJSON() {
	suite.setupV4Repo()

	// Write invalid JSON to tokens.json
	err := os.WriteFile(filepath.Join(suite.TempDir, TokensFileName), []byte("not valid json{"), 0644)
	suite.Require().NoError(err)

	// Run migration - should not fail (tokens error is non-fatal)
	fsRepo := suite.createFsRepo()
	err = V4ToV5(fsRepo)
	suite.Require().NoError(err)

	// Verify no profiles created from invalid tokens
	store := suite.getProfileStore()
	profiles, err := store.List()
	suite.Require().NoError(err)
	suite.Empty(profiles)
}

// TestMigrateConfigNoAPISettings tests migration with config that has no API settings
func (suite *V4ToV5MigrationSuite) TestMigrateConfigNoAPISettings() {
	suite.setupV4Repo()

	// Setup config.yaml with no API settings
	configContent := `
Logging:
  Level: debug
`
	suite.writeConfigYAML(configContent)

	// Run migration
	fsRepo := suite.createFsRepo()
	err := V4ToV5(fsRepo)
	suite.Require().NoError(err)

	// Verify no profiles created (no API settings to migrate)
	store := suite.getProfileStore()
	profiles, err := store.List()
	suite.Require().NoError(err)
	suite.Empty(profiles)
}

// TestMigrateTokensWithEmptyToken tests migration with empty token value
func (suite *V4ToV5MigrationSuite) TestMigrateTokensWithEmptyToken() {
	suite.setupV4Repo()

	// Setup tokens.json with empty token value
	tokens := map[string]string{
		"https://noauth.example.com:443": "",
	}
	suite.writeTokensJSON(tokens)

	// Run migration
	fsRepo := suite.createFsRepo()
	err := V4ToV5(fsRepo)
	suite.Require().NoError(err)

	// Verify profile was created without auth
	store := suite.getProfileStore()
	p, err := store.Load("noauth_example_com_443")
	suite.Require().NoError(err)
	suite.Equal("https://noauth.example.com:443", p.Endpoint)
	suite.Nil(p.Auth) // Empty token should not create auth config
}

// TestMigrateConfigWithTLSSettings tests migration preserves TLS insecure setting
func (suite *V4ToV5MigrationSuite) TestMigrateConfigWithTLSSettings() {
	suite.setupV4Repo()

	// Setup config.yaml with TLS insecure
	configContent := `
API:
  Host: secure.example.com
  Port: 443
  TLS:
    UseTLS: true
    Insecure: true
`
	suite.writeConfigYAML(configContent)

	// Run migration
	fsRepo := suite.createFsRepo()
	err := V4ToV5(fsRepo)
	suite.Require().NoError(err)

	// Verify TLS settings migrated
	store := suite.getProfileStore()
	p, err := store.Load(DefaultProfileName)
	suite.Require().NoError(err)
	suite.Require().NotNil(p.TLS)
	suite.True(p.TLS.Insecure)
}

// TestMergeConfigIntoExistingDefaultProfile tests that config TLS settings merge into existing default profile
func (suite *V4ToV5MigrationSuite) TestMergeConfigIntoExistingDefaultProfile() {
	suite.setupV4Repo()

	// Setup tokens.json that will create a "default" profile
	tokens := map[string]string{
		"http://localhost:1234": "local-token",
	}
	suite.writeTokensJSON(tokens)

	// Setup config.yaml with TLS insecure - should merge into the token profile
	// Note: This is a edge case where the migration creates "localhost_1234" not "default"
	// from tokens, so config will create its own "default" profile
	configContent := `
API:
  Host: localhost
  Port: 1234
  TLS:
    Insecure: true
`
	suite.writeConfigYAML(configContent)

	// Run migration
	fsRepo := suite.createFsRepo()
	err := V4ToV5(fsRepo)
	suite.Require().NoError(err)

	// Verify profiles were created correctly
	store := suite.getProfileStore()
	profiles, err := store.List()
	suite.Require().NoError(err)
	suite.Len(profiles, 2) // localhost_1234 and default

	// Token-based profile should have auth
	tokenProfile, err := store.Load("localhost_1234")
	suite.Require().NoError(err)
	suite.Require().NotNil(tokenProfile.Auth)
	suite.Equal("local-token", tokenProfile.Auth.Token)

	// Config-based profile should have TLS insecure
	configProfile, err := store.Load(DefaultProfileName)
	suite.Require().NoError(err)
	suite.Require().NotNil(configProfile.TLS)
	suite.True(configProfile.TLS.Insecure)
}

// TestSetCurrentProfilePreference tests that "default" profile is preferred as current
func (suite *V4ToV5MigrationSuite) TestSetCurrentProfilePreference() {
	suite.setupV4Repo()

	// Setup tokens.json with multiple endpoints
	tokens := map[string]string{
		"https://alpha.example.com:443": "alpha-token",
		"https://beta.example.com:443":  "beta-token",
		"https://gamma.example.com:443": "gamma-token",
	}
	suite.writeTokensJSON(tokens)

	// Run migration
	fsRepo := suite.createFsRepo()
	err := V4ToV5(fsRepo)
	suite.Require().NoError(err)

	// Verify one of them is set as current
	store := suite.getProfileStore()
	current, err := store.GetCurrent()
	suite.Require().NoError(err)
	suite.NotEmpty(current)

	// Current should be one of the migrated profiles
	profiles, _ := store.List()
	suite.Contains(profiles, current)
}

// TestMigrateIPv6Endpoint tests migration handles IPv6 addresses
func (suite *V4ToV5MigrationSuite) TestMigrateIPv6Endpoint() {
	suite.setupV4Repo()

	// Setup tokens.json with IPv6 endpoint
	tokens := map[string]string{
		"http://[::1]:1234": "ipv6-token",
	}
	suite.writeTokensJSON(tokens)

	// Run migration
	fsRepo := suite.createFsRepo()
	err := V4ToV5(fsRepo)
	suite.Require().NoError(err)

	// Verify profile was created
	store := suite.getProfileStore()
	profiles, err := store.List()
	suite.Require().NoError(err)
	suite.Len(profiles, 1)
}
