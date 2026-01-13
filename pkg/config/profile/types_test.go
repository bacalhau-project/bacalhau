//go:build unit || !integration

package profile_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/config/profile"
)

func TestProfileValidation(t *testing.T) {
	tests := []struct {
		name    string
		profile profile.Profile
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid profile",
			profile: profile.Profile{Endpoint: "https://api.example.com:443"},
			wantErr: false,
		},
		{
			name:    "missing endpoint",
			profile: profile.Profile{},
			wantErr: true,
			errMsg:  "endpoint is required",
		},
		{
			name:    "whitespace endpoint",
			profile: profile.Profile{Endpoint: "   "},
			wantErr: true,
			errMsg:  "endpoint is required",
		},
		{
			name:    "invalid timeout",
			profile: profile.Profile{Endpoint: "https://api.example.com:443", Timeout: "invalid"},
			wantErr: true,
			errMsg:  "invalid timeout",
		},
		{
			name:    "valid timeout",
			profile: profile.Profile{Endpoint: "https://api.example.com:443", Timeout: "60s"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.profile.Validate()
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestProfileGetTimeout(t *testing.T) {
	t.Run("default timeout", func(t *testing.T) {
		p := profile.Profile{Endpoint: "https://api.example.com:443"}
		require.Equal(t, profile.DefaultTimeout, p.GetTimeout())
	})

	t.Run("custom timeout", func(t *testing.T) {
		p := profile.Profile{Endpoint: "https://api.example.com:443", Timeout: "60s"}
		require.Equal(t, "60s", p.GetTimeout())
	})
}

func TestProfileGetToken(t *testing.T) {
	t.Run("nil auth config", func(t *testing.T) {
		p := profile.Profile{Endpoint: "https://api.example.com:443"}
		require.Equal(t, "", p.GetToken())
	})

	t.Run("empty token", func(t *testing.T) {
		p := profile.Profile{
			Endpoint: "https://api.example.com:443",
			Auth:     &profile.AuthConfig{},
		}
		require.Equal(t, "", p.GetToken())
	})

	t.Run("with token", func(t *testing.T) {
		p := profile.Profile{
			Endpoint: "https://api.example.com:443",
			Auth:     &profile.AuthConfig{Token: "my-secret-token"},
		}
		require.Equal(t, "my-secret-token", p.GetToken())
	})
}

func TestProfileIsInsecure(t *testing.T) {
	t.Run("nil TLS config", func(t *testing.T) {
		p := profile.Profile{Endpoint: "https://api.example.com:443"}
		require.False(t, p.IsInsecure())
	})

	t.Run("TLS config with insecure false", func(t *testing.T) {
		p := profile.Profile{
			Endpoint: "https://api.example.com:443",
			TLS:      &profile.TLSConfig{Insecure: false},
		}
		require.False(t, p.IsInsecure())
	})

	t.Run("TLS config with insecure true", func(t *testing.T) {
		p := profile.Profile{
			Endpoint: "https://api.example.com:443",
			TLS:      &profile.TLSConfig{Insecure: true},
		}
		require.True(t, p.IsInsecure())
	})
}
