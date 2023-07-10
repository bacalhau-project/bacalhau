//go:build unit || !integration

package math

import (
	real_math "math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMin(t *testing.T) {
	assert := assert.New(t)

	// Test scenario 1
	result := Min(3, 1, 5, 2)
	assert.Equal(1, result, "Min(3, 1, 5, 2) should be 1")

	// Test scenario 2
	result = Min(-5, -2, -10, -7)
	assert.Equal(-10, result, "Min(-5, -2, -10, -7) should be -10")

	// Test scenario 3
	result = Min(-5, 2, -10, 7)
	assert.Equal(-10, result, "Min(-5, 2, -10, 7) should be -10")

	// Test scenario 4
	result = Min(5)
	assert.Equal(5, result, "Min(5) should be 5")

	// Test scenario 5
	resultFloat := Min(3.14, 1.23, 5.67, 2.98)
	assert.Equal(1.23, resultFloat, "Min(3.14, 1.23, 5.67, 2.98) should be 1.23")

	// Test scenario 6
	resultString := Min("apricot", "banana", "apple", "date")
	assert.Equal("apple", resultString, "Min(\"apricot\", \"banana\", \"apple\", \"date\") should be \"apple\"")
}

func TestMax(t *testing.T) {
	assert := assert.New(t)

	// Test scenario 1
	result := Max(3, 1, 5, 2)
	assert.Equal(5, result, "Max(3, 1, 5, 2) should be 5")

	// Test scenario 2
	result = Max(-5, -2, -10, -7)
	assert.Equal(-2, result, "Max(-5, -2, -10, -7) should be -2")

	// Test scenario 3
	result = Max(-5, 2, -10, 7)
	assert.Equal(7, result, "Max(-5, 2, -10, 7) should be 7")

	// Test scenario 4
	result = Max(5)
	assert.Equal(5, result, "Max(5) should be 5")

	// Test scenario 5
	resultFloat := Max(3.14, 1.23, 5.67, 2.98)
	assert.Equal(5.67, resultFloat, "Max(3.14, 1.23, 5.67, 2.98) should be 5.67")

	// Test scenario 6
	resultString := Max("apricot", "banana", "apple", "date")
	assert.Equal("date", resultString, "Max(\"apricot\", \"banana\", \"apple\", \"date\") should be \"apple\"")
}
func TestAbs(t *testing.T) {
	// Test cases for integers
	t.Run("Integer - Positive", func(t *testing.T) {
		result := Abs(5)
		expected := 5
		assert.Equal(t, expected, result, "Abs(5) result mismatch")
	})

	t.Run("Integer - Negative", func(t *testing.T) {
		result := Abs(-5)
		expected := 5
		assert.Equal(t, expected, result, "Abs(-5) result mismatch")
	})

	t.Run("Integer - Zero", func(t *testing.T) {
		result := Abs(0)
		expected := 0
		assert.Equal(t, expected, result, "Abs(0) result mismatch")
	})

	// Test cases for floats
	t.Run("Float - Positive", func(t *testing.T) {
		result := Abs(3.14)
		expected := 3.14
		assert.Equal(t, expected, result, "Abs(3.14) result mismatch")
	})

	t.Run("Float - Negative", func(t *testing.T) {
		result := Abs(-3.14)
		expected := 3.14
		assert.Equal(t, expected, result, "Abs(-3.14) result mismatch")
	})

	t.Run("Float - Zero", func(t *testing.T) {
		result := Abs(0.0)
		expected := 0.0
		assert.Equal(t, expected, result, "Abs(0.0) result mismatch")
	})

	t.Run("Float - MaxValue", func(t *testing.T) {
		result := Abs(real_math.MaxFloat64)
		expected := real_math.MaxFloat64
		assert.Equal(t, expected, result, "Abs(math.MaxFloat64) result mismatch")
	})
}
