//go:build unit || !integration

package profile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/config/profile"
)

func TestStore(t *testing.T) {
	tempDir := t.TempDir()
	store := profile.NewStore(tempDir)

	t.Run("save and load profile", func(t *testing.T) {
		p := &profile.Profile{
			Endpoint:    "https://api.example.com:443",
			Description: "Test profile",
		}
		err := store.Save("test", p)
		require.NoError(t, err)

		loaded, err := store.Load("test")
		require.NoError(t, err)
		require.Equal(t, p.Endpoint, loaded.Endpoint)
		require.Equal(t, p.Description, loaded.Description)
	})

	t.Run("list profiles", func(t *testing.T) {
		// Save another profile
		err := store.Save("another", &profile.Profile{Endpoint: "https://other.com:443"})
		require.NoError(t, err)

		profiles, err := store.List()
		require.NoError(t, err)
		require.Len(t, profiles, 2)
		require.Contains(t, profiles, "test")
		require.Contains(t, profiles, "another")
	})

	t.Run("delete profile", func(t *testing.T) {
		err := store.Delete("another")
		require.NoError(t, err)

		profiles, err := store.List()
		require.NoError(t, err)
		require.Len(t, profiles, 1)
	})

	t.Run("load non-existent profile", func(t *testing.T) {
		_, err := store.Load("nonexistent")
		require.Error(t, err)
	})

	t.Run("set and get current", func(t *testing.T) {
		err := store.SetCurrent("test")
		require.NoError(t, err)

		current, err := store.GetCurrent()
		require.NoError(t, err)
		require.Equal(t, "test", current)
	})

	t.Run("delete current profile clears symlink", func(t *testing.T) {
		err := store.Delete("test")
		require.NoError(t, err)

		current, err := store.GetCurrent()
		require.NoError(t, err)
		require.Empty(t, current)
	})
}

func TestStoreSanitizeName(t *testing.T) {
	tempDir := t.TempDir()
	store := profile.NewStore(tempDir)

	// Test that dangerous names are sanitized
	p := &profile.Profile{Endpoint: "https://api.example.com:443"}
	err := store.Save("../dangerous", p)
	require.NoError(t, err)

	// filepath.Base("../dangerous") returns "dangerous", then ".." replacement doesn't apply
	profiles, err := store.List()
	require.NoError(t, err)
	require.Contains(t, profiles, "dangerous")
}

func TestStoreEmptyName(t *testing.T) {
	tempDir := t.TempDir()
	store := profile.NewStore(tempDir)

	p := &profile.Profile{Endpoint: "https://api.example.com:443"}

	t.Run("save with empty name fails", func(t *testing.T) {
		err := store.Save("", p)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("save with whitespace name fails", func(t *testing.T) {
		err := store.Save("   ", p)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("load with empty name fails", func(t *testing.T) {
		_, err := store.Load("")
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("delete with empty name fails", func(t *testing.T) {
		err := store.Delete("")
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("exists with empty name returns false", func(t *testing.T) {
		require.False(t, store.Exists(""))
	})

	t.Run("set current with empty name fails", func(t *testing.T) {
		err := store.SetCurrent("")
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot be empty")
	})
}

func TestStoreListEmpty(t *testing.T) {
	tempDir := t.TempDir()
	store := profile.NewStore(tempDir)

	// List on empty/non-existent directory should return empty slice
	profiles, err := store.List()
	require.NoError(t, err)
	require.Empty(t, profiles)
}

func TestStoreSetCurrentNonExistent(t *testing.T) {
	tempDir := t.TempDir()
	store := profile.NewStore(tempDir)

	err := store.SetCurrent("nonexistent")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestStoreOnlyWritesProvidedFields(t *testing.T) {
	tempDir := t.TempDir()
	store := profile.NewStore(tempDir)

	// Save profile with only endpoint
	p := &profile.Profile{Endpoint: "https://api.example.com:443"}
	err := store.Save("minimal", p)
	require.NoError(t, err)

	// Read raw file content
	content, err := os.ReadFile(filepath.Join(tempDir, "minimal.yaml"))
	require.NoError(t, err)

	// Should only contain endpoint, not timeout or other defaults
	require.Contains(t, string(content), "endpoint:")
	require.NotContains(t, string(content), "timeout:")
	require.NotContains(t, string(content), "auth:")
	require.NotContains(t, string(content), "tls:")
}
