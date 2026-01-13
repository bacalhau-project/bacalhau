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

func TestProfileSave(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BACALHAU_DIR", tempDir)

	t.Run("create new profile", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "save", "prod", "--endpoint", "https://api.example.com:443"})

		err := cmd.Execute()
		require.NoError(t, err)

		// Verify profile was created
		store := profile.NewStore(filepath.Join(tempDir, "profiles"))
		p, err := store.Load("prod")
		require.NoError(t, err)
		require.Equal(t, "https://api.example.com:443", p.Endpoint)

		// Verify output message
		output := buf.String()
		require.Contains(t, output, `Profile "prod" saved`)
	})

	t.Run("create profile with all options", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{
			"profile", "save", "full",
			"--endpoint", "https://api.example.com:443",
			"--description", "Full profile",
			"--timeout", "60s",
			"--insecure",
		})

		err := cmd.Execute()
		require.NoError(t, err)

		store := profile.NewStore(filepath.Join(tempDir, "profiles"))
		p, err := store.Load("full")
		require.NoError(t, err)
		require.Equal(t, "https://api.example.com:443", p.Endpoint)
		require.Equal(t, "Full profile", p.Description)
		require.Equal(t, "60s", p.Timeout)
		require.True(t, p.IsInsecure())
	})

	t.Run("create and select profile", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{
			"profile", "save", "selected",
			"--endpoint", "https://selected.example.com:443",
			"--select",
		})

		err := cmd.Execute()
		require.NoError(t, err)

		store := profile.NewStore(filepath.Join(tempDir, "profiles"))
		current, err := store.GetCurrent()
		require.NoError(t, err)
		require.Equal(t, "selected", current)

		// Verify output includes switch message
		output := buf.String()
		require.Contains(t, output, `Profile "selected" saved`)
		require.Contains(t, output, `Switched to profile "selected"`)
	})

	t.Run("update existing profile", func(t *testing.T) {
		store := profile.NewStore(filepath.Join(tempDir, "profiles"))
		err := store.Save("update-test", &profile.Profile{
			Endpoint: "https://old.example.com:443",
		})
		require.NoError(t, err)

		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "save", "update-test", "--description", "Updated"})

		err = cmd.Execute()
		require.NoError(t, err)

		p, err := store.Load("update-test")
		require.NoError(t, err)
		require.Equal(t, "https://old.example.com:443", p.Endpoint) // Preserved
		require.Equal(t, "Updated", p.Description)                  // New
	})

	t.Run("save without endpoint fails for new profile", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"profile", "save", "no-endpoint"})

		err := cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "endpoint is required")
	})

	t.Run("update timeout on existing profile", func(t *testing.T) {
		store := profile.NewStore(filepath.Join(tempDir, "profiles"))
		err := store.Save("timeout-test", &profile.Profile{
			Endpoint:    "https://timeout.example.com:443",
			Description: "Original description",
		})
		require.NoError(t, err)

		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "save", "timeout-test", "--timeout", "2m"})

		err = cmd.Execute()
		require.NoError(t, err)

		p, err := store.Load("timeout-test")
		require.NoError(t, err)
		require.Equal(t, "https://timeout.example.com:443", p.Endpoint)
		require.Equal(t, "Original description", p.Description)
		require.Equal(t, "2m", p.Timeout)
	})

	t.Run("invalid timeout fails", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{
			"profile", "save", "invalid-timeout",
			"--endpoint", "https://api.example.com:443",
			"--timeout", "invalid",
		})

		err := cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid timeout")
	})

	t.Run("update endpoint on existing profile", func(t *testing.T) {
		store := profile.NewStore(filepath.Join(tempDir, "profiles"))
		err := store.Save("endpoint-update", &profile.Profile{
			Endpoint:    "https://old.example.com:443",
			Description: "Keep this",
		})
		require.NoError(t, err)

		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{
			"profile", "save", "endpoint-update",
			"--endpoint", "https://new.example.com:443",
		})

		err = cmd.Execute()
		require.NoError(t, err)

		p, err := store.Load("endpoint-update")
		require.NoError(t, err)
		require.Equal(t, "https://new.example.com:443", p.Endpoint)
		require.Equal(t, "Keep this", p.Description) // Preserved
	})

	t.Run("missing profile name argument", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"profile", "save"})

		err := cmd.Execute()
		require.Error(t, err)
	})
}
