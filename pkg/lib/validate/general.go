package validate

import "reflect"

// NotNil checks if the provided value is not nil.
// Returns an error if the value is nil, using the provided message and arguments.
func NotNil(value any, msg string, args ...any) error {
	if value == nil {
		return createErrorf(msg, args...)
	}

	// Use reflection to handle cases where value is a nil pointer wrapped in an interface
	val := reflect.ValueOf(value)
	switch val.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func:
		if val.IsNil() {
			return createErrorf(msg, args...)
		}
	default:
	}
	return nil
}

// True checks if the provided condition is true.
// Returns an error if the condition is false, using the provided message and arguments.
func True(condition bool, msg string, args ...any) error {
	if !condition {
		return createErrorf(msg, args...)
	}
	return nil
}

// False checks if the provided condition is false.
// Returns an error if the condition is true, using the provided message and arguments.
func False(condition bool, msg string, args ...any) error {
	if condition {
		return createErrorf(msg, args...)
	}
	return nil
}
