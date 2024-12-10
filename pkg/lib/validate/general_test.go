//go:build unit || !integration

package validate

import "testing"

type doer struct{}

func (d doer) Do() {}

type Doer interface {
	Do()
}

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

	t.Run("NilFunc", func(t *testing.T) {
		var nilFunc func()
		err := NotNil(nilFunc, "value should not be nil")
		if err == nil {
			t.Errorf("NotNil failed: expected error for nil func")
		}
	})

	t.Run("NonNilFunc", func(t *testing.T) {
		nonNilFunc := func() {}
		err := NotNil(nonNilFunc, "value should not be nil")
		if err != nil {
			t.Errorf("NotNil failed: unexpected error for non-nil func")
		}
	})

	t.Run("NilSlice", func(t *testing.T) {
		var nilSlice []int
		err := NotNil(nilSlice, "value should not be nil")
		if err == nil {
			t.Errorf("NotNil failed: expected error for nil slice")
		}
	})

	t.Run("NonNilSlice", func(t *testing.T) {
		nonNilSlice := make([]int, 0)
		err := NotNil(nonNilSlice, "value should not be nil")
		if err != nil {
			t.Errorf("NotNil failed: unexpected error for non-nil slice")
		}
	})

	t.Run("NilMap", func(t *testing.T) {
		var nilMap map[string]int
		err := NotNil(nilMap, "value should not be nil")
		if err == nil {
			t.Errorf("NotNil failed: expected error for nil map")
		}
	})

	t.Run("NonNilMap", func(t *testing.T) {
		nonNilMap := make(map[string]int)
		err := NotNil(nonNilMap, "value should not be nil")
		if err != nil {
			t.Errorf("NotNil failed: unexpected error for non-nil map")
		}
	})

	t.Run("NilInterface", func(t *testing.T) {
		var nilInterface Doer
		err := NotNil(nilInterface, "value should not be nil")
		if err == nil {
			t.Errorf("NotNil failed: expected error for nil interface")
		}
	})

	t.Run("NonNilInterface", func(t *testing.T) {
		var nonNilInterface Doer = doer{}
		err := NotNil(nonNilInterface, "value should not be nil")
		if err != nil {
			t.Errorf("NotNil failed: unexpected error for non-nil interface")
		}
	})

	t.Run("FormattedMessage", func(t *testing.T) {
		err := NotNil(nil, "value %s should not be nil", "test")
		if err == nil {
			t.Errorf("NotNil failed: expected error for nil value with formatted message")
		}
		if err.Error() != "value test should not be nil" {
			t.Errorf("NotNil failed: unexpected error message, got %q", err.Error())
		}
	})
}
