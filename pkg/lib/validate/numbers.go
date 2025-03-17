package validate

import (
	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
)

// IsGreaterThanZero checks if the provided numeric value (of type T) is greater than zero.
// It returns an error if the value is not greater than zero, using the provided message and arguments.
// T is a generic type constrained to math.Number, allowing the function to work with various numeric types.
func IsGreaterThanZero[T math.Number](value T, msg string, args ...any) error {
	return IsGreaterThan(value, 0, msg, args...)
}

// IsGreaterOrEqualToZero checks if the provided numeric value (of type T) is greater or equal to zero.
// It returns an error if the value is less than zero, using the provided message and arguments.
// T is a generic type constrained to math.Number, allowing the function to work with various numeric types.
func IsGreaterOrEqualToZero[T math.Number](value T, msg string, args ...any) error {
	return IsGreaterOrEqual(value, 0, msg, args...)
}

// IsGreaterThan checks if the first provided numeric value (of type T) is greater than the second.
// It returns an error if the first value is not greater than the second, using the provided message and arguments.
// T is a generic type constrained to math.Number, allowing the function to work with various numeric types.
func IsGreaterThan[T math.Number](value, other T, msg string, args ...any) error {
	if value <= other {
		return createError(msg, args...)
	}
	return nil
}

// IsGreaterOrEqual checks if the first provided numeric value (of type T) is greater or equal to the second.
// It returns an error if the first value is less than the second, using the provided message and arguments.
// T is a generic type constrained to math.Number, allowing the function to work with various numeric types.
func IsGreaterOrEqual[T math.Number](value, other T, msg string, args ...any) error {
	if value < other {
		return createError(msg, args...)
	}
	return nil
}

// IsLessThan checks if the first provided numeric value (of type T) is less than the second.
// It returns an error if the first value is not less than the second, using the provided message and arguments.
// T is a generic type constrained to math.Number, allowing the function to work with various numeric types.
func IsLessThan[T math.Number](value, other T, msg string, args ...any) error {
	if value >= other {
		return createError(msg, args...)
	}
	return nil
}

// IsLessOrEqual checks if the first provided numeric value (of type T) is less or equal to the second.
// It returns an error if the first value is not less or equal to the second, using the provided message and arguments.
// T is a generic type constrained to math.Number, allowing the function to work with various numeric types.
func IsLessOrEqual[T math.Number](value, other T, msg string, args ...any) error {
	if value > other {
		return createError(msg, args...)
	}
	return nil
}
