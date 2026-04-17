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

func TestProfileSelect(t *testing.T) {
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

	t.Run("select profile", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "select", "prod"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, `Switched to profile "prod"`)

		// Verify current is now prod
		current, err := store.GetCurrent()
		require.NoError(t, err)
		require.Equal(t, "prod", current)
	})

	t.Run("select different profile", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "select", "dev"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, `Switched to profile "dev"`)

		// Verify current is now dev
		current, err := store.GetCurrent()
		require.NoError(t, err)
		require.Equal(t, "dev", current)
	})

	t.Run("select same profile again", func(t *testing.T) {
		// First ensure we're on dev
		err := store.SetCurrent("dev")
		require.NoError(t, err)

		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "select", "dev"})

		err = cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, `Switched to profile "dev"`)
	})

	t.Run("select non-existent fails", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"profile", "select", "nonexistent"})

		err := cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})

	t.Run("select without name fails", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"profile", "select"})

		err := cmd.Execute()
		require.Error(t, err)
	})
}
