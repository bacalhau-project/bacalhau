package util

import (
	"testing"
)

// ============================================================================
// UintValue Tests
// ============================================================================

func TestUintValue_SetValidNumber(t *testing.T) {
	var value uint64
	uv := NewUintValue(0, &value)

	err := uv.Set("123")
	if err != nil {
		t.Errorf("Expected no error for valid number, got: %v", err)
	}
	if value != 123 {
		t.Errorf("Expected value to be 123, got: %d", value)
	}
}

func TestUintValue_SetZero(t *testing.T) {
	var value uint64
	uv := NewUintValue(0, &value)

	err := uv.Set("0")
	if err != nil {
		t.Errorf("Expected no error for zero, got: %v", err)
	}
	if value != 0 {
		t.Errorf("Expected value to be 0, got: %d", value)
	}
}

func TestUintValue_SetMaxUint64(t *testing.T) {
	var value uint64
	uv := NewUintValue(0, &value)

	err := uv.Set("18446744073709551615")
	if err != nil {
		t.Errorf("Expected no error for max uint64, got: %v", err)
	}
	if value != 18446744073709551615 {
		t.Errorf("Expected value to be max uint64, got: %d", value)
	}
}

func TestUintValue_SetNegativeNumber(t *testing.T) {
	var value uint64
	uv := NewUintValue(0, &value)

	err := uv.Set("-1")
	if err == nil {
		t.Error("Expected error for negative number, got nil")
	}
	expectedMsg := "'-1' is not a valid number: please provide a positive integer"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got: %s", expectedMsg, err.Error())
	}
}

func TestUintValue_SetFloat(t *testing.T) {
	var value uint64
	uv := NewUintValue(0, &value)

	err := uv.Set("12.34")
	if err == nil {
		t.Error("Expected error for float number, got nil")
	}
	expectedMsg := "'12.34' is not a valid number: please provide a positive integer"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got: %s", expectedMsg, err.Error())
	}
}

func TestUintValue_SetNonNumeric(t *testing.T) {
	var value uint64
	uv := NewUintValue(0, &value)

	err := uv.Set("abc")
	if err == nil {
		t.Error("Expected error for non-numeric string, got nil")
	}
	expectedMsg := "'abc' is not a valid number: please provide a positive integer"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got: %s", expectedMsg, err.Error())
	}
}

func TestUintValue_SetEmptyString(t *testing.T) {
	var value uint64
	uv := NewUintValue(0, &value)

	err := uv.Set("")
	if err == nil {
		t.Error("Expected error for empty string, got nil")
	}
	expectedMsg := "'' is not a valid number: please provide a positive integer"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got: %s", expectedMsg, err.Error())
	}
}

func TestUintValue_Type(t *testing.T) {
	var value uint64
	uv := NewUintValue(0, &value)

	if uv.Type() != "uint" {
		t.Errorf("Expected Type() to return 'uint', got: %s", uv.Type())
	}
}

func TestUintValue_String(t *testing.T) {
	var value uint64
	uv := NewUintValue(42, &value)

	if uv.String() != "42" {
		t.Errorf("Expected String() to return '42', got: %s", uv.String())
	}
}

func TestUintValue_StringAfterSet(t *testing.T) {
	var value uint64
	uv := NewUintValue(0, &value)
	uv.Set("999")

	if uv.String() != "999" {
		t.Errorf("Expected String() to return '999', got: %s", uv.String())
	}
}

func TestNewUintValue_InitialValue(t *testing.T) {
	var value uint64
	uv := NewUintValue(100, &value)

	if value != 100 {
		t.Errorf("Expected initial value to be 100, got: %d", value)
	}
	if uv.String() != "100" {
		t.Errorf("Expected String() to return '100', got: %s", uv.String())
	}
}
