//go:build unit || !integration

package validate

import (
	"testing"
)

func TestIsGreaterThanZero(t *testing.T) {
	// Test with value less than zero
	err := IsGreaterThanZero(-1, "value should be greater than zero")
	if err == nil || err.Error() != "value should be greater than zero" {
		t.Errorf("IsGreaterThanZero failed: expected error for value -1")
	}

	// Test with zero
	err = IsGreaterThanZero(0, "value should be greater than zero")
	if err == nil || err.Error() != "value should be greater than zero" {
		t.Errorf("IsGreaterThanZero failed: expected error for value 0")
	}

	// Test with value greater than zero
	err = IsGreaterThanZero(1, "value should be greater than zero")
	if err != nil {
		t.Errorf("IsGreaterThanZero failed: unexpected error for value 1")
	}

	// Test with different numeric types
	var floatValue float64 = 1.5
	err = IsGreaterThanZero(floatValue, "value should be greater than zero")
	if err != nil {
		t.Errorf("IsGreaterThanZero failed: unexpected error for float value %v", floatValue)
	}
}
