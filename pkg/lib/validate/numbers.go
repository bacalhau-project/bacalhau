package validate

import (
	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
)

// IsGreaterThanZero checks if the provided numeric value (of type T) is greater than zero.
// It returns an error if the value is not greater than zero, using the provided message and arguments.
// T is a generic type constrained to math.Number, allowing the function to work with various numeric types.
func IsGreaterThanZero[T math.Number](value T, msg string, args ...any) error {
	if value <= 0 {
		return createError(msg, args...)
	}
	return nil
}

// IsGreaterOrEqualToZero checks if the provided numeric value (of type T) is greater or equal to zero.
// It returns an error if the value is less than zero, using the provided message and arguments.
// T is a generic type constrained to math.Number, allowing the function to work with various numeric types.
func IsGreaterOrEqualToZero[T math.Number](value T, msg string, args ...any) error {
	if value < 0 {
		return createError(msg, args...)
	}
	return nil
}
