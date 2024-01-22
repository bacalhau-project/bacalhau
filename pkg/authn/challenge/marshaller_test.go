//go:build unit || !integration

package challenge

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestStringMarshaller_MarshalBinary tests the MarshalBinary method of StringMarshaller.
func TestStringMarshaller_MarshalBinary(t *testing.T) {
	testCases := []struct {
		input string
	}{
		{"hello"},
		{""},
		{"12345"},
	}

	for _, tc := range testCases {
		m := NewStringMarshaller(tc.input)
		marshaled, err := m.MarshalBinary()
		require.NoError(t, err, "MarshalBinary() with input %s returned an unexpected error", tc.input)

		// Manually unmarshal and compare with the original input
		require.Equal(t, []byte(tc.input), marshaled, "MarshalBinary() with input %s returned an unexpected byte slice", tc.input)
	}
}
