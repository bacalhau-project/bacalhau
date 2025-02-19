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

func TestIsGreaterOrEqualToZero(t *testing.T) {
	// Test with value less than zero
	err := IsGreaterOrEqualToZero(-1, "value should be greater or equal to zero")
	if err == nil || err.Error() != "value should be greater or equal to zero" {
		t.Errorf("IsGreaterOrEqualToZero failed: expected error for value -1")
	}

	// Test with zero
	err = IsGreaterOrEqualToZero(0, "value should be greater or equal to zero")
	if err != nil {
		t.Errorf("IsGreaterOrEqualToZero failed: unexpected error for value 0")
	}

	// Test with value greater than zero
	err = IsGreaterOrEqualToZero(1, "value should be greater or equal to zero")
	if err != nil {
		t.Errorf("IsGreaterOrEqualToZero failed: unexpected error for value 1")
	}

	// Test with different numeric types
	var floatValue float64 = 0.0
	err = IsGreaterOrEqualToZero(floatValue, "value should be greater or equal to zero")
	if err != nil {
		t.Errorf("IsGreaterOrEqualToZero failed: unexpected error for float value %v", floatValue)
	}
}

func TestIsGreaterThan(t *testing.T) {
	// Test with value less than other
	err := IsGreaterThan(1, 2, "value should be greater than other")
	if err == nil || err.Error() != "value should be greater than other" {
		t.Errorf("IsGreaterThan failed: expected error for values 1, 2")
	}

	// Test with value equal to other
	err = IsGreaterThan(2, 2, "value should be greater than other")
	if err == nil || err.Error() != "value should be greater than other" {
		t.Errorf("IsGreaterThan failed: expected error for values 2, 2")
	}

	// Test with value greater than other
	err = IsGreaterThan(3, 2, "value should be greater than other")
	if err != nil {
		t.Errorf("IsGreaterThan failed: unexpected error for values 3, 2")
	}

	// Test with different numeric types
	var floatValue float64 = 2.5
	var otherFloatValue float64 = 1.5
	err = IsGreaterThan(floatValue, otherFloatValue, "value should be greater than other")
	if err != nil {
		t.Errorf("IsGreaterThan failed: unexpected error for float values %v, %v", floatValue, otherFloatValue)
	}
}

func TestIsGreaterOrEqual(t *testing.T) {
	// Test with value less than other
	err := IsGreaterOrEqual(1, 2, "value should be greater or equal to other")
	if err == nil || err.Error() != "value should be greater or equal to other" {
		t.Errorf("IsGreaterOrEqual failed: expected error for values 1, 2")
	}

	// Test with value equal to other
	err = IsGreaterOrEqual(2, 2, "value should be greater or equal to other")
	if err != nil {
		t.Errorf("IsGreaterOrEqual failed: unexpected error for values 2, 2")
	}

	// Test with value greater than other
	err = IsGreaterOrEqual(3, 2, "value should be greater or equal to other")
	if err != nil {
		t.Errorf("IsGreaterOrEqual failed: unexpected error for values 3, 2")
	}

	// Test with different numeric types
	var floatValue float64 = 1.5
	var otherFloatValue float64 = 1.5
	err = IsGreaterOrEqual(floatValue, otherFloatValue, "value should be greater or equal to other")
	if err != nil {
		t.Errorf("IsGreaterOrEqual failed: unexpected error for float values %v, %v", floatValue, otherFloatValue)
	}
}

func TestIsLessThan(t *testing.T) {
	// Test with value greater than other
	err := IsLessThan(2, 1, "value should be less than other")
	if err == nil || err.Error() != "value should be less than other" {
		t.Errorf("IsLessThan failed: expected error for values 2, 1")
	}

	// Test with value equal to other
	err = IsLessThan(2, 2, "value should be less than other")
	if err == nil || err.Error() != "value should be less than other" {
		t.Errorf("IsLessThan failed: expected error for values 2, 2")
	}

	// Test with value less than other
	err = IsLessThan(1, 2, "value should be less than other")
	if err != nil {
		t.Errorf("IsLessThan failed: unexpected error for values 1, 2")
	}

	// Test with different numeric types
	var floatValue float64 = 1.5
	var otherFloatValue float64 = 2.5
	err = IsLessThan(floatValue, otherFloatValue, "value should be less than other")
	if err != nil {
		t.Errorf("IsLessThan failed: unexpected error for float values %v, %v", floatValue, otherFloatValue)
	}
}

func TestIsLessOrEqual(t *testing.T) {
	// Test with value greater than other
	err := IsLessOrEqual(2, 1, "value should be less or equal to other")
	if err == nil || err.Error() != "value should be less or equal to other" {
		t.Errorf("IsLessOrEqual failed: expected error for values 2, 1")
	}

	// Test with value equal to other
	err = IsLessOrEqual(2, 2, "value should be less or equal to other")
	if err != nil {
		t.Errorf("IsLessOrEqual failed: unexpected error for values 2, 2")
	}

	// Test with value less than other
	err = IsLessOrEqual(1, 2, "value should be less or equal to other")
	if err != nil {
		t.Errorf("IsLessOrEqual failed: unexpected error for values 1, 2")
	}

	// Test with different numeric types
	var floatValue float64 = 1.5
	var otherFloatValue float64 = 1.5
	err = IsLessOrEqual(floatValue, otherFloatValue, "value should be less or equal to other")
	if err != nil {
		t.Errorf("IsLessOrEqual failed: unexpected error for float values %v, %v", floatValue, otherFloatValue)
	}
}
