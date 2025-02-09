//go:build unit || !integration

package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
)

type EnvVarValueSuite struct {
	suite.Suite
}

func TestEnvVarValueSuite(t *testing.T) {
	suite.Run(t, new(EnvVarValueSuite))
}

// TestSerialization ensures EnvVarValue can be properly serialized/deserialized
// cspell:ignore pecial
func (s *EnvVarValueSuite) TestSerialization() {
	tests := []struct {
		name     string
		value    EnvVarValue
		wantJSON string
	}{
		{
			name:     "literal value",
			value:    "simple-value",
			wantJSON: `"simple-value"`,
		},
		{
			name:     "env reference",
			value:    "env:TEST_VAR",
			wantJSON: `"env:TEST_VAR"`,
		},
		{
			name:     "empty value",
			value:    "",
			wantJSON: `""`,
		},
		{
			name:     "value with special chars",
			value:    "value with spaces and $pecial ch@rs",
			wantJSON: `"value with spaces and $pecial ch@rs"`,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Test JSON marshaling
			got, err := json.Marshal(tt.value)
			s.NoError(err)
			s.JSONEq(tt.wantJSON, string(got))

			// Test JSON unmarshaling
			var value EnvVarValue
			err = json.Unmarshal([]byte(tt.wantJSON), &value)
			s.NoError(err)
			s.Equal(tt.value, value)
		})
	}
}

func (s *EnvVarValueSuite) TestValidation() {
	tests := []struct {
		name      string
		env       map[string]EnvVarValue
		shouldErr bool
	}{
		{
			name: "valid env vars",
			env: map[string]EnvVarValue{
				"VALID_VAR": "value",
				"TEST_VAR":  "env:HOST_VAR",
			},
			shouldErr: false,
		},
		{
			name: "invalid var name with space",
			env: map[string]EnvVarValue{
				"INVALID NAME": "value",
			},
			shouldErr: true,
		},
		{
			name: "invalid var name with special chars",
			env: map[string]EnvVarValue{
				"INVALID@NAME": "value",
			},
			shouldErr: true,
		},
		{
			name: "invalid lowercase var name",
			env: map[string]EnvVarValue{
				"lowercase": "value",
			},
			shouldErr: true,
		},
		{
			name: "invalid var name starting with number",
			env: map[string]EnvVarValue{
				"1VAR": "value",
			},
			shouldErr: true,
		},
		{
			name: "reserved prefix",
			env: map[string]EnvVarValue{
				"BACALHAU_TEST": "value",
			},
			shouldErr: true,
		},
		{
			name: "multiple vars with one invalid",
			env: map[string]EnvVarValue{
				"VALID_VAR":     "value",
				"INVALID NAME":  "value",
				"ANOTHER_VALID": "value",
			},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := ValidateEnvVars(tt.env)
			if tt.shouldErr {
				s.Error(err)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *EnvVarValueSuite) TestEnvVarsToStringMap() {
	tests := []struct {
		name string
		env  map[string]EnvVarValue
		want map[string]string
	}{
		{
			name: "nil map",
			env:  nil,
			want: nil,
		},
		{
			name: "empty map",
			env:  map[string]EnvVarValue{},
			want: map[string]string{},
		},
		{
			name: "simple values",
			env: map[string]EnvVarValue{
				"KEY1": "value1",
				"KEY2": "value2",
			},
			want: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
			},
		},
		{
			name: "env references",
			env: map[string]EnvVarValue{
				"HOST_VAR": "env:TEST_VAR",
				"LITERAL":  "literal-value",
			},
			want: map[string]string{
				"HOST_VAR": "env:TEST_VAR",
				"LITERAL":  "literal-value",
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := EnvVarsToStringMap(tt.env)
			s.Equal(tt.want, got)
		})
	}
}
