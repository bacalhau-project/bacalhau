//go:build unit || !integration

package validate

import (
	"strings"
	"testing"
)

func TestCreateError(t *testing.T) {
	// Test with no arguments
	err := createErrorf("simple error")
	if err == nil || err.Error() != "simple error" {
		t.Errorf("createErrorf failed: expected 'simple error', got '%v'", err)
	}

	// Test with arguments
	err = createErrorf("error with argument: %v", 42)
	if err == nil || !strings.Contains(err.Error(), "42") {
		t.Errorf("createErrorf failed: expected string containing '42', got '%v'", err)
	}

	// Test with multiple arguments
	err = createErrorf("error with multiple arguments: %v %s", 42, "test")
	expectedMsg := "error with multiple arguments: 42 test"
	if err == nil || err.Error() != expectedMsg {
		t.Errorf("createErrorf failed: expected '%s', got '%v'", expectedMsg, err)
	}
}
