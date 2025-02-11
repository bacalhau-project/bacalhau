//go:build unit || !integration

package env

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type HostResolverSuite struct {
	suite.Suite
	resolver *HostResolver
}

func TestHostResolverSuite(t *testing.T) {
	suite.Run(t, new(HostResolverSuite))
}

func (s *HostResolverSuite) SetupTest() {
	s.resolver = NewHostResolver([]string{
		"ALLOWED_*",
		"TEST_VAR",
	})
}

func (s *HostResolverSuite) TestValidate() {
	tests := []struct {
		name      string
		varName   string
		varValue  string
		shouldErr bool
	}{
		{
			name:      "allowed pattern",
			varName:   "job_var",
			varValue:  "ALLOWED_VALUE",
			shouldErr: false,
		},
		{
			name:      "allowed exact match",
			varName:   "job_var",
			varValue:  "TEST_VAR",
			shouldErr: false,
		},
		{
			name:      "not allowed",
			varName:   "job_var",
			varValue:  "DENIED_VALUE",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := s.resolver.Validate(tt.varName, tt.varValue)
			if tt.shouldErr {
				s.Error(err)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *HostResolverSuite) TestValue() {
	// Set test environment variables
	s.T().Setenv("ALLOWED_VAR", "allowed_value")
	s.T().Setenv("TEST_VAR", "test_value")
	s.T().Setenv("DENIED_VAR", "denied_value")

	tests := []struct {
		name        string
		value       string
		expected    string
		shouldErr   bool
		errContains string
	}{
		{
			name:      "allowed pattern var exists",
			value:     "ALLOWED_VAR",
			expected:  "allowed_value",
			shouldErr: false,
		},
		{
			name:      "allowed exact match var exists",
			value:     "TEST_VAR",
			expected:  "test_value",
			shouldErr: false,
		},
		{
			name:        "not allowed var",
			value:       "DENIED_VAR",
			shouldErr:   true,
			errContains: "not in allowed",
		},
		{
			name:        "allowed pattern var doesn't exist",
			value:       "ALLOWED_MISSING",
			shouldErr:   true,
			errContains: "not found",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			val, err := s.resolver.Value(tt.value)
			if tt.shouldErr {
				s.Error(err)
				s.Contains(err.Error(), tt.errContains)
			} else {
				s.NoError(err)
				s.Equal(tt.expected, val)
			}
		})
	}
}
