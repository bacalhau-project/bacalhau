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

// ============================================================================
// BoolValue Tests
// ============================================================================

func TestBoolValue_SetTrue(t *testing.T) {
	var value bool
	bv := NewBoolValue(false, &value)

	err := bv.Set("true")
	if err != nil {
		t.Errorf("Expected no error for 'true', got: %v", err)
	}
	if !value {
		t.Error("Expected value to be true")
	}
}

func TestBoolValue_SetFalse(t *testing.T) {
	var value bool
	bv := NewBoolValue(true, &value)

	err := bv.Set("false")
	if err != nil {
		t.Errorf("Expected no error for 'false', got: %v", err)
	}
	if value {
		t.Error("Expected value to be false")
	}
}

func TestBoolValue_SetOne(t *testing.T) {
	var value bool
	bv := NewBoolValue(false, &value)

	err := bv.Set("1")
	if err != nil {
		t.Errorf("Expected no error for '1', got: %v", err)
	}
	if !value {
		t.Error("Expected value to be true for '1'")
	}
}

func TestBoolValue_SetZero(t *testing.T) {
	var value bool
	bv := NewBoolValue(true, &value)

	err := bv.Set("0")
	if err != nil {
		t.Errorf("Expected no error for '0', got: %v", err)
	}
	if value {
		t.Error("Expected value to be false for '0'")
	}
}

func TestBoolValue_SetT(t *testing.T) {
	var value bool
	bv := NewBoolValue(false, &value)

	err := bv.Set("t")
	if err != nil {
		t.Errorf("Expected no error for 't', got: %v", err)
	}
	if !value {
		t.Error("Expected value to be true for 't'")
	}
}

func TestBoolValue_SetF(t *testing.T) {
	var value bool
	bv := NewBoolValue(true, &value)

	err := bv.Set("f")
	if err != nil {
		t.Errorf("Expected no error for 'f', got: %v", err)
	}
	if value {
		t.Error("Expected value to be false for 'f'")
	}
}

func TestBoolValue_SetUpperT(t *testing.T) {
	var value bool
	bv := NewBoolValue(false, &value)

	err := bv.Set("T")
	if err != nil {
		t.Errorf("Expected no error for 'T', got: %v", err)
	}
	if !value {
		t.Error("Expected value to be true for 'T'")
	}
}

func TestBoolValue_SetUpperF(t *testing.T) {
	var value bool
	bv := NewBoolValue(true, &value)

	err := bv.Set("F")
	if err != nil {
		t.Errorf("Expected no error for 'F', got: %v", err)
	}
	if value {
		t.Error("Expected value to be false for 'F'")
	}
}

func TestBoolValue_SetTRUE(t *testing.T) {
	var value bool
	bv := NewBoolValue(false, &value)

	err := bv.Set("TRUE")
	if err != nil {
		t.Errorf("Expected no error for 'TRUE', got: %v", err)
	}
	if !value {
		t.Error("Expected value to be true for 'TRUE'")
	}
}

func TestBoolValue_SetFALSE(t *testing.T) {
	var value bool
	bv := NewBoolValue(true, &value)

	err := bv.Set("FALSE")
	if err != nil {
		t.Errorf("Expected no error for 'FALSE', got: %v", err)
	}
	if value {
		t.Error("Expected value to be false for 'FALSE'")
	}
}

func TestBoolValue_SetTrue_MixedCase(t *testing.T) {
	var value bool
	bv := NewBoolValue(false, &value)

	err := bv.Set("True")
	if err != nil {
		t.Errorf("Expected no error for 'True', got: %v", err)
	}
	if !value {
		t.Error("Expected value to be true for 'True'")
	}
}

func TestBoolValue_SetFalse_MixedCase(t *testing.T) {
	var value bool
	bv := NewBoolValue(true, &value)

	err := bv.Set("False")
	if err != nil {
		t.Errorf("Expected no error for 'False', got: %v", err)
	}
	if value {
		t.Error("Expected value to be false for 'False'")
	}
}

func TestBoolValue_SetInvalidString(t *testing.T) {
	var value bool
	bv := NewBoolValue(false, &value)

	err := bv.Set("maybe")
	if err == nil {
		t.Error("Expected error for 'maybe', got nil")
	}
	expectedMsg := "'maybe' is not a valid boolean: please provide 'true' or 'false'"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got: %s", expectedMsg, err.Error())
	}
}

func TestBoolValue_SetYes(t *testing.T) {
	var value bool
	bv := NewBoolValue(false, &value)

	err := bv.Set("yes")
	if err == nil {
		t.Error("Expected error for 'yes', got nil")
	}
	expectedMsg := "'yes' is not a valid boolean: please provide 'true' or 'false'"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got: %s", expectedMsg, err.Error())
	}
}

func TestBoolValue_SetNo(t *testing.T) {
	var value bool
	bv := NewBoolValue(false, &value)

	err := bv.Set("no")
	if err == nil {
		t.Error("Expected error for 'no', got nil")
	}
	expectedMsg := "'no' is not a valid boolean: please provide 'true' or 'false'"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got: %s", expectedMsg, err.Error())
	}
}

func TestBoolValue_SetTwo(t *testing.T) {
	var value bool
	bv := NewBoolValue(false, &value)

	err := bv.Set("2")
	if err == nil {
		t.Error("Expected error for '2', got nil")
	}
	expectedMsg := "'2' is not a valid boolean: please provide 'true' or 'false'"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got: %s", expectedMsg, err.Error())
	}
}

func TestBoolValue_SetEmptyString(t *testing.T) {
	var value bool
	bv := NewBoolValue(false, &value)

	err := bv.Set("")
	if err == nil {
		t.Error("Expected error for empty string, got nil")
	}
	expectedMsg := "'' is not a valid boolean: please provide 'true' or 'false'"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got: %s", expectedMsg, err.Error())
	}
}

func TestBoolValue_Type(t *testing.T) {
	var value bool
	bv := NewBoolValue(false, &value)

	if bv.Type() != "bool" {
		t.Errorf("Expected Type() to return 'bool', got: %s", bv.Type())
	}
}

func TestBoolValue_StringTrue(t *testing.T) {
	var value bool
	bv := NewBoolValue(true, &value)

	if bv.String() != "true" {
		t.Errorf("Expected String() to return 'true', got: %s", bv.String())
	}
}

func TestBoolValue_StringFalse(t *testing.T) {
	var value bool
	bv := NewBoolValue(false, &value)

	if bv.String() != "false" {
		t.Errorf("Expected String() to return 'false', got: %s", bv.String())
	}
}

func TestBoolValue_StringAfterSet(t *testing.T) {
	var value bool
	bv := NewBoolValue(false, &value)
	bv.Set("true")

	if bv.String() != "true" {
		t.Errorf("Expected String() to return 'true' after setting, got: %s", bv.String())
	}
}

func TestNewBoolValue_InitialValueTrue(t *testing.T) {
	var value bool
	bv := NewBoolValue(true, &value)

	if !value {
		t.Error("Expected initial value to be true")
	}
	if bv.String() != "true" {
		t.Errorf("Expected String() to return 'true', got: %s", bv.String())
	}
}

func TestNewBoolValue_InitialValueFalse(t *testing.T) {
	var value bool
	bv := NewBoolValue(false, &value)

	if value {
		t.Error("Expected initial value to be false")
	}
	if bv.String() != "false" {
		t.Errorf("Expected String() to return 'false', got: %s", bv.String())
	}
}
