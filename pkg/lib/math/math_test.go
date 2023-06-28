//go:build unit || !integration

package math

import (
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
