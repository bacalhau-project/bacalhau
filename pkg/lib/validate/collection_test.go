//go:build unit || !integration

package validate

import (
	"errors"
	"testing"
)

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		msg      string
		expected error
	}{
		{"Empty int slice", []int{}, "Int slice is not empty", nil},
		{"Non-empty int slice", []int{1, 2, 3}, "Int slice is not empty", errors.New("Int slice is not empty")},
		{"Empty string slice", []string{}, "String slice is not empty", nil},
		{"Non-empty string slice", []string{"a", "b"}, "String slice is not empty", errors.New("String slice is not empty")},
		{"Empty struct slice", []struct{}{}, "Struct slice is not empty", nil},
		{"Non-empty struct slice", []struct{}{{}}, "Struct slice is not empty", errors.New("Struct slice is not empty")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			switch v := tt.input.(type) {
			case []int:
				err = IsEmpty(v, tt.msg)
			case []string:
				err = IsEmpty(v, tt.msg)
			case []struct{}:
				err = IsEmpty(v, tt.msg)
			}

			if (err == nil && tt.expected != nil) || (err != nil && tt.expected == nil) {
				t.Errorf("IsEmpty() error = %v, expected %v", err, tt.expected)
			}
			if err != nil && tt.expected != nil && err.Error() != tt.expected.Error() {
				t.Errorf("IsEmpty() error message = %v, expected %v", err, tt.expected)
			}
		})
	}
}

func TestIsNotEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		msg      string
		expected error
	}{
		{"Empty int slice", []int{}, "Int slice is empty", errors.New("Int slice is empty")},
		{"Non-empty int slice", []int{1, 2, 3}, "Int slice is empty", nil},
		{"Empty string slice", []string{}, "String slice is empty", errors.New("String slice is empty")},
		{"Non-empty string slice", []string{"a", "b"}, "String slice is empty", nil},
		{"Empty struct slice", []struct{}{}, "Struct slice is empty", errors.New("Struct slice is empty")},
		{"Non-empty struct slice", []struct{}{{}}, "Struct slice is empty", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			switch v := tt.input.(type) {
			case []int:
				err = IsNotEmpty(v, tt.msg)
			case []string:
				err = IsNotEmpty(v, tt.msg)
			case []struct{}:
				err = IsNotEmpty(v, tt.msg)
			}

			if (err == nil && tt.expected != nil) || (err != nil && tt.expected == nil) {
				t.Errorf("IsNotEmpty() error = %v, expected %v", err, tt.expected)
			}
			if err != nil && tt.expected != nil && err.Error() != tt.expected.Error() {
				t.Errorf("IsNotEmpty() error message = %v, expected %v", err, tt.expected)
			}
		})
	}
}

func TestKeyNotInMap(t *testing.T) {
	// Test with string keys and int values
	t.Run("String to Int Map", func(t *testing.T) {
		myMap := map[string]int{"a": 1, "b": 2, "c": 3}

		// Test with a key that doesn't exist
		err := KeyNotInMap("d", myMap, "key should not exist in map")
		if err != nil {
			t.Errorf("KeyNotInMap failed: unexpected error for non-existent key")
		}

		// Test with a key that exists
		err = KeyNotInMap("a", myMap, "key should not exist in map")
		if err == nil || err.Error() != "key should not exist in map" {
			t.Errorf("KeyNotInMap failed: expected error for existing key")
		}
	})

	// Test with int keys and string values
	t.Run("Int to String Map", func(t *testing.T) {
		myMap := map[int]string{1: "one", 2: "two", 3: "three"}

		// Test with a key that doesn't exist
		err := KeyNotInMap(4, myMap, "key should not exist in map")
		if err != nil {
			t.Errorf("KeyNotInMap failed: unexpected error for non-existent key")
		}

		// Test with a key that exists
		err = KeyNotInMap(1, myMap, "key should not exist in map")
		if err == nil || err.Error() != "key should not exist in map" {
			t.Errorf("KeyNotInMap failed: expected error for existing key")
		}
	})

	// Test with an empty map
	t.Run("Empty Map", func(t *testing.T) {
		emptyMap := make(map[string]int)
		err := KeyNotInMap("a", emptyMap, "key should not exist in map")
		if err != nil {
			t.Errorf("KeyNotInMap failed: unexpected error for empty map")
		}
	})

	// Test with a nil map
	t.Run("Nil Map", func(t *testing.T) {
		var nilMap map[string]int
		err := KeyNotInMap("a", nilMap, "key should not exist in map")
		if err != nil {
			t.Errorf("KeyNotInMap failed: unexpected error for nil map")
		}
	})

	// Test with custom struct as value
	t.Run("Map with Struct Value", func(t *testing.T) {
		type CustomStruct struct {
			Value int
		}
		myMap := map[string]CustomStruct{"a": {Value: 1}, "b": {Value: 2}}

		// Test with a key that doesn't exist
		err := KeyNotInMap("c", myMap, "key should not exist in map")
		if err != nil {
			t.Errorf("KeyNotInMap failed: unexpected error for non-existent key")
		}

		// Test with a key that exists
		err = KeyNotInMap("a", myMap, "key should not exist in map")
		if err == nil || err.Error() != "key should not exist in map" {
			t.Errorf("KeyNotInMap failed: expected error for existing key")
		}
	})

	// Test with custom error message and arguments
	t.Run("Custom Error Message", func(t *testing.T) {
		myMap := map[string]int{"a": 1, "b": 2, "c": 3}
		key := "a"
		err := KeyNotInMap(key, myMap, "key %s should not exist in map", key)
		if err == nil || err.Error() != "key a should not exist in map" {
			t.Errorf("KeyNotInMap failed: unexpected error message for custom error")
		}
	})
}
