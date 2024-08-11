//go:build unit || !integration

package validate

import (
	"testing"
)

func TestNotBlank(t *testing.T) {
	// Test with empty string
	err := NotBlank("", "string should not be blank")
	if err == nil || err.Error() != "string should not be blank" {
		t.Errorf("NotBlank failed: expected error for empty string")
	}

	// Test with whitespace-only string
	err = NotBlank("   ", "string should not be blank")
	if err == nil || err.Error() != "string should not be blank" {
		t.Errorf("NotBlank failed: expected error for whitespace-only string")
	}

	// Test with non-blank string
	err = NotBlank("hello", "string should not be blank")
	if err != nil {
		t.Errorf("NotBlank failed: unexpected error for non-blank string")
	}
}

func TestNoSpaces(t *testing.T) {
	// Test with string containing spaces
	err := NoSpaces("hello world", "string should not contain spaces")
	if err == nil || err.Error() != "string should not contain spaces" {
		t.Errorf("NoSpaces failed: expected error for string with spaces")
	}

	// Test with string containing tabs
	err = NoSpaces("hello\tworld", "string should not contain spaces")
	if err == nil || err.Error() != "string should not contain spaces" {
		t.Errorf("NoSpaces failed: expected error for string with tabs")
	}

	// Test with string without spaces
	err = NoSpaces("helloworld", "string should not contain spaces")
	if err != nil {
		t.Errorf("NoSpaces failed: unexpected error for string without spaces")
	}
}

func TestNoNullChars(t *testing.T) {
	// Test with string containing null character
	err := NoNullChars("hello\x00world", "string should not contain null characters")
	if err == nil || err.Error() != "string should not contain null characters" {
		t.Errorf("NoNullChars failed: expected error for string with null character")
	}

	// Test with string without null characters
	err = NoNullChars("helloworld", "string should not contain null characters")
	if err != nil {
		t.Errorf("NoNullChars failed: unexpected error for string without null characters")
	}
}

func TestContainsNoneOf(t *testing.T) {
	// Test with string containing multiple specified characters
	err := ContainsNoneOf("hello123", "123", "string should not contain specified characters")
	if err == nil || err.Error() != "string should not contain specified characters" {
		t.Errorf("ContainsNoneOf failed: expected error for string containing multiple specified characters")
	}

	// Test with string containing only one specified character
	err = ContainsNoneOf("hello1world", "123", "string should not contain specified characters")
	if err == nil || err.Error() != "string should not contain specified characters" {
		t.Errorf("ContainsNoneOf failed: expected error for string containing one specified character")
	}

	// Test with string containing only the last specified character
	err = ContainsNoneOf("hello3world", "123", "string should not contain specified characters")
	if err == nil || err.Error() != "string should not contain specified characters" {
		t.Errorf("ContainsNoneOf failed: expected error for string containing the last specified character")
	}

	// Test with string not containing any specified characters
	err = ContainsNoneOf("helloworld", "123", "string should not contain specified characters")
	if err != nil {
		t.Errorf("ContainsNoneOf failed: unexpected error for string not containing specified characters")
	}

	// Test with empty string for both input and specified characters
	err = ContainsNoneOf("", "", "string should not contain specified characters")
	if err != nil {
		t.Errorf("ContainsNoneOf failed: unexpected error for empty string and empty specified characters")
	}
}
