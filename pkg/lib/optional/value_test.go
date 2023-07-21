//go:build unit || !integration

package optional

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValueOptional_IsPresent(t *testing.T) {
	value := "Hello, World!"
	opt := New(value) // Create an instance of ValueOptional with a value
	isPresent := opt.IsPresent()
	assert.True(t, isPresent, "Expected IsPresent to be true for a non-empty optional")
}

func TestValueOptional_Value(t *testing.T) {
	value := 42
	opt := New(value) // Create an instance of ValueOptional with a value
	result, err := opt.Get()
	assert.NoError(t, err, "Expected no error when calling Get on a non-empty optional")
	assert.Equal(t, value, result, "Expected value to match the stored value in the optional")
}

func TestValueOptional_GetValueOrDefault(t *testing.T) {
	value := true
	defaultValue := false
	opt := New(value) // Create an instance of ValueOptional with a value
	result := opt.GetOrDefault(defaultValue)
	assert.Equal(t, value, result, "Expected GetOrDefault to return the stored value in the optional")
}
