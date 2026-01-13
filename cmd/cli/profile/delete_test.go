//go:build unit || !integration

package profile_test

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	cli "github.com/bacalhau-project/bacalhau/cmd/cli"
	"github.com/bacalhau-project/bacalhau/pkg/config/profile"
)

func TestProfileDelete(t *testing.T) {
	t.Run("delete profile", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("BACALHAU_DIR", tempDir)

		// Setup profile
		store := profile.NewStore(filepath.Join(tempDir, "profiles"))
		err := store.Save("test", &profile.Profile{
			Endpoint: "https://test.example.com:443",
		})
		require.NoError(t, err)

		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "delete", "test"})

		err = cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, `Profile "test" deleted`)

		// Verify profile no longer exists
		require.False(t, store.Exists("test"))
	})

	t.Run("delete with force", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("BACALHAU_DIR", tempDir)

		// Setup profile and set as current
		store := profile.NewStore(filepath.Join(tempDir, "profiles"))
		err := store.Save("current-test", &profile.Profile{
			Endpoint: "https://current.example.com:443",
		})
		require.NoError(t, err)
		err = store.SetCurrent("current-test")
		require.NoError(t, err)

		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "delete", "current-test", "--force"})

		err = cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, `Profile "current-test" deleted`)

		// Verify profile no longer exists
		require.False(t, store.Exists("current-test"))
	})

	t.Run("delete with short force flag", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("BACALHAU_DIR", tempDir)

		// Setup profile and set as current
		store := profile.NewStore(filepath.Join(tempDir, "profiles"))
		err := store.Save("short-force", &profile.Profile{
			Endpoint: "https://short.example.com:443",
		})
		require.NoError(t, err)
		err = store.SetCurrent("short-force")
		require.NoError(t, err)

		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "delete", "short-force", "-f"})

		err = cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, `Profile "short-force" deleted`)
	})

	t.Run("delete non-existent fails", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("BACALHAU_DIR", tempDir)

		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"profile", "delete", "nonexistent"})

		err := cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})

	t.Run("delete without name fails", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("BACALHAU_DIR", tempDir)

		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"profile", "delete"})

		err := cmd.Execute()
		require.Error(t, err)
	})

	t.Run("delete current profile with confirmation yes", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("BACALHAU_DIR", tempDir)

		// Setup profile and set as current
		store := profile.NewStore(filepath.Join(tempDir, "profiles"))
		err := store.Save("confirm-test", &profile.Profile{
			Endpoint: "https://confirm.example.com:443",
		})
		require.NoError(t, err)
		err = store.SetCurrent("confirm-test")
		require.NoError(t, err)

		cmd := cli.NewRootCmd()
		outBuf := new(bytes.Buffer)
		inBuf := strings.NewReader("y\n")
		cmd.SetOut(outBuf)
		cmd.SetIn(inBuf)
		cmd.SetArgs([]string{"profile", "delete", "confirm-test"})

		err = cmd.Execute()
		require.NoError(t, err)

		output := outBuf.String()
		require.Contains(t, output, `Profile "confirm-test" deleted`)

		// Verify profile no longer exists
		require.False(t, store.Exists("confirm-test"))
	})

	t.Run("delete current profile with confirmation no", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("BACALHAU_DIR", tempDir)

		// Setup profile and set as current
		store := profile.NewStore(filepath.Join(tempDir, "profiles"))
		err := store.Save("cancel-test", &profile.Profile{
			Endpoint: "https://cancel.example.com:443",
		})
		require.NoError(t, err)
		err = store.SetCurrent("cancel-test")
		require.NoError(t, err)

		cmd := cli.NewRootCmd()
		outBuf := new(bytes.Buffer)
		inBuf := strings.NewReader("n\n")
		cmd.SetOut(outBuf)
		cmd.SetIn(inBuf)
		cmd.SetArgs([]string{"profile", "delete", "cancel-test"})

		err = cmd.Execute()
		require.NoError(t, err)

		output := outBuf.String()
		require.Contains(t, output, "Deletion cancelled")

		// Verify profile still exists
		require.True(t, store.Exists("cancel-test"))
	})

	t.Run("delete non-current profile without prompt", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("BACALHAU_DIR", tempDir)

		// Setup two profiles
		store := profile.NewStore(filepath.Join(tempDir, "profiles"))
		err := store.Save("keep", &profile.Profile{
			Endpoint: "https://keep.example.com:443",
		})
		require.NoError(t, err)
		err = store.Save("delete-me", &profile.Profile{
			Endpoint: "https://delete.example.com:443",
		})
		require.NoError(t, err)

		// Set "keep" as current
		err = store.SetCurrent("keep")
		require.NoError(t, err)

		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "delete", "delete-me"})

		err = cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, `Profile "delete-me" deleted`)
		// Should not contain confirmation prompt
		require.NotContains(t, output, "Delete anyway")

		// Verify profile was deleted
		require.False(t, store.Exists("delete-me"))
		// Verify other profile still exists
		require.True(t, store.Exists("keep"))
	})
}
