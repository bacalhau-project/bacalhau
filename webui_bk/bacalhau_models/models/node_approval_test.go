//go:build unit || !integration

package models

import (
	"reflect"
	"testing"
)

func TestTrimQuotes(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{
			name:     "with surrounding quotes",
			input:    []byte(`"example"`),
			expected: []byte(`example`),
		},
		{
			name:     "with escaped quotes inside",
			input:    []byte(`"ex\"ample"`),
			expected: []byte(`ex\"ample`),
		},
		{
			name:     "without surrounding quotes",
			input:    []byte(`example`),
			expected: []byte(`example`),
		},
		{
			name:     "with only one quote at the beginning",
			input:    []byte(`"example`),
			expected: []byte(`"example`),
		},
		{
			name:     "with only one quote at the end",
			input:    []byte(`example"`),
			expected: []byte(`example"`),
		},
		{
			name:     "empty byte slice",
			input:    []byte(``),
			expected: []byte(``),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := trimQuotes(tt.input); !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("trimQuotes() = %v, want %v", got, tt.expected)
			}
		})
	}
}
