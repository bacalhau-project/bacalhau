//go:build unit || !integration

package env

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ResolverMapSuite struct {
	suite.Suite
	resolver *ResolverMap
}

func TestResolverMapSuite(t *testing.T) {
	suite.Run(t, new(ResolverMapSuite))
}

func (s *ResolverMapSuite) SetupTest() {
	s.resolver = NewResolver(ResolverParams{
		AllowList: []string{"TEST_*", "ALLOWED_*"},
	})
}

func (s *ResolverMapSuite) TestParseValue() {
	tests := []struct {
		name         string
		value        string
		expectPrefix string
		expectRest   string
	}{
		{
			name:         "valid env reference",
			value:        "env:TEST_VAR",
			expectPrefix: "env",
			expectRest:   "TEST_VAR",
		},
		{
			name:         "literal value",
			value:        "literal",
			expectPrefix: "",
			expectRest:   "literal",
		},
		{
			name:         "empty value",
			value:        "",
			expectPrefix: "",
			expectRest:   "",
		},
		{
			name:         "only prefix",
			value:        "env:",
			expectPrefix: "env",
			expectRest:   "",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			prefix, rest := parseValue(tt.value)
			s.Equal(tt.expectPrefix, prefix)
			s.Equal(tt.expectRest, rest)
		})
	}
}

func (s *ResolverMapSuite) TestValidate() {
	tests := []struct {
		name      string
		varName   string
		value     string
		shouldErr bool
	}{
		{
			name:      "valid env reference",
			varName:   "job_var",
			value:     "env:TEST_VAR",
			shouldErr: false,
		},
		{
			name:      "literal value",
			varName:   "job_var",
			value:     "literal",
			shouldErr: false,
		},
		{
			name:      "not allowed env var",
			varName:   "job_var",
			value:     "env:DENIED_VAR",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := s.resolver.Validate(tt.varName, tt.value)
			if tt.shouldErr {
				s.Error(err)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *ResolverMapSuite) TestValue() {
	// Set test environment variables
	s.T().Setenv("TEST_VAR", "test_value")
	s.T().Setenv("ALLOWED_VAR", "allowed_value")

	tests := []struct {
		name        string
		value       string
		expected    string
		shouldErr   bool
		errContains string
	}{
		{
			name:      "valid env reference",
			value:     "env:TEST_VAR",
			expected:  "test_value",
			shouldErr: false,
		},
		{
			name:      "literal value",
			value:     "literal",
			expected:  "literal",
			shouldErr: false,
		},
		{
			name:        "not allowed env var",
			value:       "env:DENIED_VAR",
			shouldErr:   true,
			errContains: "not in allowed",
		},
		{
			name:        "env var doesn't exist",
			value:       "env:TEST_MISSING",
			shouldErr:   true,
			errContains: "not found",
		},
		{
			name:      "unknown prefix treated as literal",
			value:     "unknown:value",
			expected:  "unknown:value",
			shouldErr: false,
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
