//go:build unit || !integration

package profile_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	cli "github.com/bacalhau-project/bacalhau/cmd/cli"
	"github.com/bacalhau-project/bacalhau/pkg/config/profile"
)

// TestProfileWorkflow tests the complete profile workflow:
// save -> list -> show -> select -> delete
func TestProfileWorkflow(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BACALHAU_DIR", tempDir)

	// Step 1: Save a new profile
	t.Run("1. save new profile", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{
			"profile", "save", "production",
			"--endpoint", "https://prod.example.com:443",
			"--description", "Production cluster",
			"--timeout", "60s",
		})

		err := cmd.Execute()
		require.NoError(t, err)
		require.Contains(t, buf.String(), "production")
	})

	// Step 2: Save another profile with --select
	t.Run("2. save second profile with --select", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{
			"profile", "save", "staging",
			"--endpoint", "https://staging.example.com:443",
			"--insecure",
			"--select",
		})

		err := cmd.Execute()
		require.NoError(t, err)

		// Verify staging is now current
		store := profile.NewStore(filepath.Join(tempDir, "profiles"))
		current, err := store.GetCurrent()
		require.NoError(t, err)
		require.Equal(t, "staging", current)
	})

	// Step 3: List profiles
	t.Run("3. list profiles", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "list"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, "production")
		require.Contains(t, output, "staging")
		require.Contains(t, output, "https://prod.example.com:443")
		require.Contains(t, output, "https://staging.example.com:443")
		// Staging should have * marker as current
		require.Contains(t, output, "*")
	})

	// Step 4: Show production profile
	t.Run("4. show production profile", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "show", "production"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, "Name:        production")
		require.Contains(t, output, "Endpoint:    https://prod.example.com:443")
		require.Contains(t, output, "Description: Production cluster")
		require.Contains(t, output, "Timeout:     60s")
	})

	// Step 5: Show staging profile (should show insecure)
	t.Run("5. show staging profile with insecure", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "show", "staging"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, "staging (current)")
		require.Contains(t, output, "TLS:         insecure")
	})

	// Step 6: Select production profile
	t.Run("6. select production profile", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "select", "production"})

		err := cmd.Execute()
		require.NoError(t, err)

		// Verify production is now current
		store := profile.NewStore(filepath.Join(tempDir, "profiles"))
		current, err := store.GetCurrent()
		require.NoError(t, err)
		require.Equal(t, "production", current)
	})

	// Step 7: Show current profile (should be production)
	t.Run("7. show current profile", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "show"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, "production (current)")
	})

	// Step 8: Update existing profile
	t.Run("8. update existing profile", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{
			"profile", "save", "staging",
			"--description", "Updated staging description",
		})

		err := cmd.Execute()
		require.NoError(t, err)

		// Verify update
		store := profile.NewStore(filepath.Join(tempDir, "profiles"))
		p, err := store.Load("staging")
		require.NoError(t, err)
		require.Equal(t, "Updated staging description", p.Description)
		// Original values should be preserved
		require.Equal(t, "https://staging.example.com:443", p.Endpoint)
		require.True(t, p.IsInsecure())
	})

	// Step 9: Delete staging profile (without --force should fail in non-interactive)
	t.Run("9. delete staging profile with --force", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "delete", "staging", "--force"})

		err := cmd.Execute()
		require.NoError(t, err)

		// Verify staging is deleted
		store := profile.NewStore(filepath.Join(tempDir, "profiles"))
		require.False(t, store.Exists("staging"))
	})

	// Step 10: List should only show production
	t.Run("10. list after delete", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "list"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, "production")
		require.NotContains(t, output, "staging")
	})
}

// TestProfileWithAuth tests profile workflow with authentication
func TestProfileWithAuth(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BACALHAU_DIR", tempDir)

	// Save profile with auth token
	store := profile.NewStore(filepath.Join(tempDir, "profiles"))
	err := store.Save("secure", &profile.Profile{
		Endpoint:    "https://secure.example.com:443",
		Description: "Secure cluster",
		Auth: &profile.AuthConfig{
			Token: "super-secret-token-12345678",
		},
	})
	require.NoError(t, err)
	err = store.SetCurrent("secure")
	require.NoError(t, err)

	t.Run("token is redacted by default", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "show"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, "token (supe****5678)")
		require.NotContains(t, output, "super-secret-token-12345678")
	})

	t.Run("--show-token reveals full token", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "show", "--show-token"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, "super-secret-token-12345678")
	})

	t.Run("list shows auth status", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "list"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, "token")
	})
}

// TestProfileOutputFormats tests different output formats
func TestProfileOutputFormats(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BACALHAU_DIR", tempDir)

	// Setup profiles
	store := profile.NewStore(filepath.Join(tempDir, "profiles"))
	err := store.Save("test", &profile.Profile{
		Endpoint:    "https://test.example.com:443",
		Description: "Test profile",
	})
	require.NoError(t, err)

	t.Run("list JSON format", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "list", "--output", "json"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, `"name":"test"`)
		require.Contains(t, output, `"endpoint":"https://test.example.com:443"`)
	})

	t.Run("list YAML format", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "list", "--output", "yaml"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, "name: test")
		require.Contains(t, output, "endpoint: https://test.example.com:443")
	})
}

// TestProfileErrorHandling tests error cases
func TestProfileErrorHandling(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BACALHAU_DIR", tempDir)

	t.Run("show non-existent profile", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"profile", "show", "nonexistent"})

		err := cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})

	t.Run("select non-existent profile", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"profile", "select", "nonexistent"})

		err := cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})

	t.Run("delete non-existent profile", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"profile", "delete", "nonexistent", "--force"})

		err := cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})

	t.Run("save without endpoint on new profile", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"profile", "save", "newprofile"})

		err := cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "endpoint")
	})

	t.Run("save with empty profile name", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"profile", "save"})

		err := cmd.Execute()
		require.Error(t, err)
	})

	t.Run("show without current profile set", func(t *testing.T) {
		// No profiles exist yet
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"profile", "show"})

		err := cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "no current profile")
	})
}

// TestMigrationIntegration tests that migration works with profiles
func TestMigrationIntegration(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BACALHAU_DIR", tempDir)

	// Simulate a migrated profile scenario
	store := profile.NewStore(filepath.Join(tempDir, "profiles"))

	// Create profiles as migration would
	err := store.Save("localhost_1234", &profile.Profile{
		Endpoint:    "http://localhost:1234",
		Description: "Migrated from tokens.json",
		Auth: &profile.AuthConfig{
			Token: "migrated-token",
		},
	})
	require.NoError(t, err)

	err = store.Save("default", &profile.Profile{
		Endpoint:    "http://localhost:1234",
		Description: "Migrated from config.yaml",
	})
	require.NoError(t, err)

	err = store.SetCurrent("default")
	require.NoError(t, err)

	t.Run("list migrated profiles", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "list"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, "localhost_1234")
		require.Contains(t, output, "default")
	})

	t.Run("show migrated profile with token", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "show", "localhost_1234"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, "Migrated from tokens.json")
		require.Contains(t, output, "token (")
	})
}
