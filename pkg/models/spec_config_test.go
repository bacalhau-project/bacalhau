//go:build unit || !integration

package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSpecConfigNormalize(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]interface{}
		want   map[string]interface{}
	}{
		{
			name:   "nil map",
			params: nil,
			want:   nil,
		},
		{
			name:   "empty map",
			params: map[string]interface{}{},
			want:   nil,
		},
		{
			name:   "non-empty map",
			params: map[string]interface{}{"key": "value"},
			want:   map[string]interface{}{"key": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SpecConfig{Params: tt.params}
			s.Normalize()
			if (s.Params == nil) != (tt.want == nil) {
				t.Errorf("SpecConfig.Normalize() = %v, want %v", s.Params, tt.want)
			}
		})
	}
}

func TestSpecConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		Type    string
		wantErr bool
	}{
		{
			name:    "empty type",
			Type:    "",
			wantErr: true,
		},
		{
			name:    "non-empty type",
			Type:    "test",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SpecConfig{Type: tt.Type}
			if err := s.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("SpecConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSpecConfigNull(t *testing.T) {
	var s *SpecConfig
	s.Normalize()
	require.Error(t, s.Validate())
}
