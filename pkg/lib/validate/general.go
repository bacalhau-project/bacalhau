package validate

import "reflect"

// IsNotNil checks if the provided value is not nil.
// Returns an error if the value is nil, using the provided message and arguments.
func IsNotNil(value any, msg string, args ...any) error {
	if value == nil {
		return createError(msg, args...)
	}

	// Use reflection to handle cases where value is a nil pointer wrapped in an interface
	val := reflect.ValueOf(value)
	if val.Kind() == reflect.Ptr && val.IsNil() {
		return createError(msg, args...)
	}
	return nil
}
