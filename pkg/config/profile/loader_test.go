//go:build unit || !integration

package profile_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/config/profile"
)

func TestLoader(t *testing.T) {
	tempDir := t.TempDir()
	store := profile.NewStore(tempDir)

	// Setup: create profiles
	err := store.Save("prod", &profile.Profile{
		Endpoint:    "https://prod.example.com:443",
		Description: "Production",
	})
	require.NoError(t, err)

	err = store.Save("dev", &profile.Profile{
		Endpoint: "https://dev.example.com:443",
	})
	require.NoError(t, err)

	err = store.SetCurrent("prod")
	require.NoError(t, err)

	t.Run("load current profile", func(t *testing.T) {
		loader := profile.NewLoader(store, "", "")
		p, name, err := loader.Load()
		require.NoError(t, err)
		require.Equal(t, "prod", name)
		require.Equal(t, "https://prod.example.com:443", p.Endpoint)
	})

	t.Run("flag overrides current", func(t *testing.T) {
		loader := profile.NewLoader(store, "dev", "")
		p, name, err := loader.Load()
		require.NoError(t, err)
		require.Equal(t, "dev", name)
		require.Equal(t, "https://dev.example.com:443", p.Endpoint)
	})

	t.Run("env var overrides current", func(t *testing.T) {
		loader := profile.NewLoader(store, "", "dev")
		p, name, err := loader.Load()
		require.NoError(t, err)
		require.Equal(t, "dev", name)
		require.Equal(t, "https://dev.example.com:443", p.Endpoint)
	})

	t.Run("flag overrides env var", func(t *testing.T) {
		loader := profile.NewLoader(store, "prod", "dev")
		p, name, err := loader.Load()
		require.NoError(t, err)
		require.Equal(t, "prod", name)
		require.Equal(t, "https://prod.example.com:443", p.Endpoint)
	})

	t.Run("no profile returns nil", func(t *testing.T) {
		emptyStore := profile.NewStore(t.TempDir())
		loader := profile.NewLoader(emptyStore, "", "")
		p, name, err := loader.Load()
		require.NoError(t, err)
		require.Nil(t, p)
		require.Empty(t, name)
	})

	t.Run("non-existent profile flag errors", func(t *testing.T) {
		loader := profile.NewLoader(store, "nonexistent", "")
		_, _, err := loader.Load()
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})
}

func TestLoaderLoadOrCreate(t *testing.T) {
	t.Run("creates new profile", func(t *testing.T) {
		tempDir := t.TempDir()
		store := profile.NewStore(tempDir)
		loader := profile.NewLoader(store, "", "")

		p, err := loader.LoadOrCreate("new-profile", "https://api.example.com:443")
		require.NoError(t, err)
		require.NotNil(t, p)
		require.Equal(t, "https://api.example.com:443", p.Endpoint)

		// Verify it was actually saved
		require.True(t, store.Exists("new-profile"))
		loaded, err := store.Load("new-profile")
		require.NoError(t, err)
		require.Equal(t, "https://api.example.com:443", loaded.Endpoint)
	})

	t.Run("loads existing profile", func(t *testing.T) {
		tempDir := t.TempDir()
		store := profile.NewStore(tempDir)

		// Create existing profile with extra fields
		existing := &profile.Profile{
			Endpoint:    "https://existing.example.com:443",
			Description: "Existing profile",
			Auth:        &profile.AuthConfig{Token: "existing-token"},
		}
		err := store.Save("existing", existing)
		require.NoError(t, err)

		loader := profile.NewLoader(store, "", "")
		p, err := loader.LoadOrCreate("existing", "https://different.example.com:443")
		require.NoError(t, err)
		require.NotNil(t, p)
		// Should return existing profile, not create new one
		require.Equal(t, "https://existing.example.com:443", p.Endpoint)
		require.Equal(t, "Existing profile", p.Description)
		require.Equal(t, "existing-token", p.GetToken())
	})

	t.Run("created profile has only endpoint", func(t *testing.T) {
		tempDir := t.TempDir()
		store := profile.NewStore(tempDir)
		loader := profile.NewLoader(store, "", "")

		p, err := loader.LoadOrCreate("minimal", "https://api.example.com:443")
		require.NoError(t, err)
		require.Equal(t, "https://api.example.com:443", p.Endpoint)
		require.Empty(t, p.Description)
		require.Empty(t, p.Timeout)
		require.Nil(t, p.Auth)
		require.Nil(t, p.TLS)
	})
}
