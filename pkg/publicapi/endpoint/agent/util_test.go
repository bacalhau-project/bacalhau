package agent

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/stretchr/testify/require"
)

func TestRedactConfigSensitiveInfo(t *testing.T) {
	originalConfig := types.Bacalhau{
		API: types.API{
			Auth: types.AuthConfig{
				Users: []types.AuthUser{
					{
						Username: "testuser",
						Password: "secretpassword",
						APIKey:   "secretapikey",
						Alias:    "testalias",
						Capabilities: []types.Capability{
							{Actions: []string{"read", "write"}},
						},
					},
					{
						Username: "user2",
						Password: "",
						APIKey:   "anotherapikey",
					},
				},
			},
		},
	}

	redactedConfig, err := redactConfigSensitiveInfo(originalConfig)
	require.NoError(t, err)

	// Verify sensitive information is redacted
	require.Equal(t, "********", redactedConfig.API.Auth.Users[0].Password)
	require.Equal(t, "********", redactedConfig.API.Auth.Users[0].APIKey)
	require.Equal(t, "********", redactedConfig.API.Auth.Users[1].APIKey)

	// Verify non-sensitive information remains unchanged
	require.Equal(t, "testuser", redactedConfig.API.Auth.Users[0].Username)
	require.Equal(t, "testalias", redactedConfig.API.Auth.Users[0].Alias)
	require.Equal(t, []string{"read", "write"}, redactedConfig.API.Auth.Users[0].Capabilities[0].Actions)

	// Verify empty values remain empty
	require.Empty(t, redactedConfig.API.Auth.Users[1].Password)

	// Verify original config is not modified
	require.Equal(t, "secretpassword", originalConfig.API.Auth.Users[0].Password)
	require.Equal(t, "secretapikey", originalConfig.API.Auth.Users[0].APIKey)
	require.Equal(t, "anotherapikey", originalConfig.API.Auth.Users[1].APIKey)
}
