package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResourceMemory(t *testing.T) {
	tests := []struct {
		in  string
		exp uint64
	}{
		{"42", 42},          // 42 Bytes
		{"42MB", 42000000},  // 42 Megabytes
		{"42MiB", 44040192}, // 42 Mebibyte
		{"42mb", 42000000},  // 42 Megabytes
		{"42mib", 44040192}, // 42 Mebibyte
	}

	for _, p := range tests {
		cfg, err := NewResourcesConfigBuilder().Memory(p.in).Build()
		require.NoError(t, err)
		// TODO I think this is where the bug occurs
		actual, err := cfg.ToResources()
		require.NoError(t, err)

		require.Equal(t, p.exp, actual.Memory)
	}
}
