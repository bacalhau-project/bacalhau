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

func TestProfileList(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BACALHAU_DIR", tempDir)

	// Setup profiles
	store := profile.NewStore(filepath.Join(tempDir, "profiles"))
	err := store.Save("prod", &profile.Profile{
		Endpoint:    "https://prod.example.com:443",
		Description: "Production",
		Auth:        &profile.AuthConfig{Token: "secret"},
	})
	require.NoError(t, err)

	err = store.Save("dev", &profile.Profile{
		Endpoint: "http://localhost:1234",
	})
	require.NoError(t, err)

	err = store.SetCurrent("prod")
	require.NoError(t, err)

	t.Run("list profiles table format", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "list"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, "prod")
		require.Contains(t, output, "dev")
		require.Contains(t, output, "https://prod.example.com:443")
	})

	t.Run("list profiles json format", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "list", "--output", "json"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, `"name":"prod"`)
		require.Contains(t, output, `"name":"dev"`)
	})

	t.Run("list profiles yaml format", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "list", "--output", "yaml"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, "name: prod")
		require.Contains(t, output, "name: dev")
	})

	t.Run("current profile marker", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "list"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		// Current profile (prod) should have * marker
		require.Contains(t, output, "*")
	})

	t.Run("auth token display", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "list"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		// Profile with token should show "token" in AUTH column
		require.Contains(t, output, "token")
	})
}

func TestProfileListEmpty(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BACALHAU_DIR", tempDir)

	cmd := cli.NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"profile", "list"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	require.Contains(t, output, "No profiles found")
}
