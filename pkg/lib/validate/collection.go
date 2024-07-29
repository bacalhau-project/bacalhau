package validate

// IsEmpty checks if the provided slice is empty.
// It returns an error if the slice is not empty, using the provided message and arguments.
// T is a generic type, allowing the function to work with slices of any type.
func IsEmpty[T any](s []T, msg string, args ...any) error {
	if len(s) != 0 {
		return createError(msg, args...)
	}
	return nil
}

// IsNotEmpty checks if the provided slice is not empty.
// It returns an error if the slice is empty, using the provided message and arguments.
// T is a generic type, allowing the function to work with slices of any type.
func IsNotEmpty[T any](s []T, msg string, args ...any) error {
	if len(s) == 0 {
		return createError(msg, args...)
	}
	return nil
}

// KeyNotInMap checks if the given key does not exist in the provided map.
// It returns an error if the key exists in the map, using the provided message and arguments.
// K is a generic type for the map key, which must be comparable.
// V is a generic type for the map value, which can be any type.
func KeyNotInMap[K comparable, V any](key K, m map[K]V, msg string, args ...any) error {
	if _, exists := m[key]; exists {
		return createError(msg, args...)
	}
	return nil
}
