//go:build unit || !integration

package optional

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmptyOptional_IsPresent(t *testing.T) {
	opt := Empty[string]()
	isPresent := opt.IsPresent()
	assert.False(t, isPresent, "Expected IsPresent to be false for an empty optional")
}

func TestEmptyOptional_Value(t *testing.T) {
	opt := Empty[string]()
	value, err := opt.Get()
	assert.Error(t, err, "Expected an error when calling Get on an empty optional")
	assert.Equal(t, "", value, "Expected value to be nil when calling Get on an empty optional")
}

func TestEmptyOptional_GetValueOrDefault(t *testing.T) {
	opt := Empty[int]()
	defaultValue := 42
	result := opt.GetOrDefault(defaultValue)
	assert.Equal(t, defaultValue, result, "Expected GetOrDefault to return the default value for an empty optional")
}
