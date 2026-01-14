package profile

import (
	"fmt"
	"strings"
	"time"
)

const (
	// DefaultTimeout is the default request timeout for profile connections.
	DefaultTimeout = "30s"
)

// Profile represents a CLI connection profile for a Bacalhau cluster.
type Profile struct {
	// Endpoint is the API endpoint (host:port or full URL). Required.
	Endpoint string `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
	// Description is an optional user-friendly label.
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	// Timeout is the request timeout as a duration string (e.g., "30s").
	Timeout string `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	// Auth contains authentication settings.
	Auth *AuthConfig `yaml:"auth,omitempty" json:"auth,omitempty"`
	// TLS contains TLS/SSL settings.
	TLS *TLSConfig `yaml:"tls,omitempty" json:"tls,omitempty"`
}

// AuthConfig contains authentication settings for a profile.
type AuthConfig struct {
	// Token is the bearer token for API authentication.
	Token string `yaml:"token,omitempty" json:"token,omitempty"`
}

// TLSConfig contains TLS settings for a profile.
type TLSConfig struct {
	// Insecure skips TLS certificate verification.
	Insecure bool `yaml:"insecure,omitempty" json:"insecure,omitempty"`
}

// Validate validates the profile configuration.
func (p *Profile) Validate() error {
	if strings.TrimSpace(p.Endpoint) == "" {
		return fmt.Errorf("endpoint is required")
	}
	if p.Timeout != "" {
		if _, err := time.ParseDuration(p.Timeout); err != nil {
			return fmt.Errorf("invalid timeout %q: %w", p.Timeout, err)
		}
	}
	return nil
}

// GetTimeout returns the timeout duration string, or the default if not set.
// Note: This method assumes Validate() has already been called. The returned
// timeout string is not re-validated and may be invalid if Validate() was skipped.
func (p *Profile) GetTimeout() string {
	if p.Timeout == "" {
		return DefaultTimeout
	}
	return p.Timeout
}

// GetToken returns the auth token if set, or empty string.
func (p *Profile) GetToken() string {
	if p.Auth == nil {
		return ""
	}
	return p.Auth.Token
}

// IsInsecure returns whether TLS verification should be skipped.
func (p *Profile) IsInsecure() bool {
	if p.TLS == nil {
		return false
	}
	return p.TLS.Insecure
}
