package reflection

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStructName(t *testing.T) {
	tests := []struct {
		name     string
		subject  any
		expected string
	}{
		{
			name:     "handles-pointers",
			subject:  &foo{},
			expected: "pkg/util/reflection.foo",
		},
		{
			name:     "handles-bare",
			subject:  foo{},
			expected: "pkg/util/reflection.foo",
		},
		{
			name:     "ignore-non-bacalhau-prefix",
			subject:  assert.New(t),
			expected: "github.com/stretchr/testify/assert.Assertions",
		},
		{
			name:     "does-not-crash-with-non-structs",
			subject:  123,
			expected: "int",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := StructName(test.subject)
			assert.Equal(t, test.expected, actual)
		})
	}
}

type foo struct {
}
