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

func TestGlobalProfileFlag(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BACALHAU_DIR", tempDir)

	// Setup profiles
	store := profile.NewStore(filepath.Join(tempDir, "profiles"))
	err := store.Save("prod", &profile.Profile{
		Endpoint: "https://prod.example.com:443",
	})
	require.NoError(t, err)
	err = store.Save("dev", &profile.Profile{
		Endpoint: "http://localhost:1234",
	})
	require.NoError(t, err)
	err = store.SetCurrent("prod")
	require.NoError(t, err)

	t.Run("explicit arg takes precedence over current profile", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		// Current is prod, but we use explicit "dev" arg
		cmd.SetArgs([]string{"profile", "show", "dev"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		// Should show dev profile, not prod
		require.Contains(t, output, "Name:        dev")
		require.Contains(t, output, "http://localhost:1234")
		require.NotContains(t, output, "https://prod.example.com:443")
	})

	t.Run("without flag uses current profile", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "show"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		// Should show prod (current) profile
		require.Contains(t, output, "prod (current)")
		require.Contains(t, output, "https://prod.example.com:443")
	})

	t.Run("invalid profile name errors", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"profile", "show", "nonexistent"})

		err := cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})
}

func TestProfileEnvVar(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BACALHAU_DIR", tempDir)

	// Setup profiles
	store := profile.NewStore(filepath.Join(tempDir, "profiles"))
	err := store.Save("prod", &profile.Profile{
		Endpoint: "https://prod.example.com:443",
	})
	require.NoError(t, err)
	err = store.Save("dev", &profile.Profile{
		Endpoint: "http://localhost:1234",
	})
	require.NoError(t, err)
	err = store.SetCurrent("prod")
	require.NoError(t, err)

	t.Run("BACALHAU_PROFILE env var", func(t *testing.T) {
		t.Setenv("BACALHAU_PROFILE", "dev")

		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "show"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, "dev")
		require.Contains(t, output, "http://localhost:1234")
	})

	t.Run("explicit arg takes precedence over env var", func(t *testing.T) {
		t.Setenv("BACALHAU_PROFILE", "dev")

		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		// Explicit arg specifies prod, env var specifies dev
		cmd.SetArgs([]string{"profile", "show", "prod"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		// Explicit arg should take precedence
		require.Contains(t, output, "prod")
		require.Contains(t, output, "https://prod.example.com:443")
	})

	t.Run("env var takes precedence over current profile", func(t *testing.T) {
		// Set current to prod but env to dev
		t.Setenv("BACALHAU_PROFILE", "dev")

		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "show"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		// Env var should override current profile
		require.Contains(t, output, "dev")
		require.Contains(t, output, "http://localhost:1234")
	})

	t.Run("invalid env var profile errors", func(t *testing.T) {
		t.Setenv("BACALHAU_PROFILE", "nonexistent")

		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"profile", "show"})

		err := cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})
}

func TestProfilePrecedence(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BACALHAU_DIR", tempDir)

	// Setup profiles
	store := profile.NewStore(filepath.Join(tempDir, "profiles"))
	err := store.Save("current-profile", &profile.Profile{
		Endpoint: "https://current.example.com:443",
	})
	require.NoError(t, err)
	err = store.Save("env-profile", &profile.Profile{
		Endpoint: "https://env.example.com:443",
	})
	require.NoError(t, err)
	err = store.Save("flag-profile", &profile.Profile{
		Endpoint: "https://flag.example.com:443",
	})
	require.NoError(t, err)
	err = store.SetCurrent("current-profile")
	require.NoError(t, err)

	t.Run("explicit arg takes precedence over env and current", func(t *testing.T) {
		// Set env var
		t.Setenv("BACALHAU_PROFILE", "env-profile")

		// With explicit arg, env, and current all set, explicit arg wins
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "show", "flag-profile"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, "flag-profile")
		require.Contains(t, output, "https://flag.example.com:443")
	})

	t.Run("env var takes precedence over current when no explicit arg", func(t *testing.T) {
		// Set env var
		t.Setenv("BACALHAU_PROFILE", "env-profile")

		// With env and current set but no explicit arg, env wins
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "show"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, "env-profile")
		require.Contains(t, output, "https://env.example.com:443")
	})
}
