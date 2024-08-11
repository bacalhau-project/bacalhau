//go:build unit || !integration

package validate

import "testing"

// TestIsNotNil tests the NotNil function for various scenarios.
func TestIsNotNil(t *testing.T) {
	t.Run("NilValue", func(t *testing.T) {
		err := NotNil(nil, "value should not be nil")
		if err == nil {
			t.Errorf("NotNil failed: expected error for nil value")
		}
	})

	t.Run("NonNilValue", func(t *testing.T) {
		err := NotNil(42, "value should not be nil")
		if err != nil {
			t.Errorf("NotNil failed: unexpected error for non-nil value")
		}
	})

	t.Run("NilPointer", func(t *testing.T) {
		var nilPointer *int
		err := NotNil(nilPointer, "value should not be nil")
		if err == nil {
			t.Errorf("NotNil failed: expected error for nil pointer")
		}
	})

	t.Run("NonNilPointer", func(t *testing.T) {
		nonNilPointer := new(int)
		err := NotNil(nonNilPointer, "value should not be nil")
		if err != nil {
			t.Errorf("NotNil failed: unexpected error for non-nil pointer")
		}
	})
}
