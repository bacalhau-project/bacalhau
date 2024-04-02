//go:build unit || !integration

package validate

import "testing"

// TestIsNotNil tests the IsNotNil function for various scenarios.
func TestIsNotNil(t *testing.T) {
	t.Run("NilValue", func(t *testing.T) {
		err := IsNotNil(nil, "value should not be nil")
		if err == nil {
			t.Errorf("IsNotNil failed: expected error for nil value")
		}
	})

	t.Run("NonNilValue", func(t *testing.T) {
		err := IsNotNil(42, "value should not be nil")
		if err != nil {
			t.Errorf("IsNotNil failed: unexpected error for non-nil value")
		}
	})

	t.Run("NilPointer", func(t *testing.T) {
		var nilPointer *int
		err := IsNotNil(nilPointer, "value should not be nil")
		if err == nil {
			t.Errorf("IsNotNil failed: expected error for nil pointer")
		}
	})

	t.Run("NonNilPointer", func(t *testing.T) {
		nonNilPointer := new(int)
		err := IsNotNil(nonNilPointer, "value should not be nil")
		if err != nil {
			t.Errorf("IsNotNil failed: unexpected error for non-nil pointer")
		}
	})
}
