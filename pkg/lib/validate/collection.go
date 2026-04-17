package validate

// IsEmpty checks if the provided slice is empty.
// It returns an error if the slice is not empty, using the provided message.
func IsEmpty[T any](s []T, msg string) error {
	if len(s) != 0 {
		return createError(msg)
	}
	return nil
}

// IsEmptyf checks if the provided slice is empty with formatted message.
// It returns an error if the slice is not empty, formatting the message with the provided arguments.
func IsEmptyf[T any](s []T, format string, args ...any) error {
	if len(s) != 0 {
		return createErrorf(format, args...)
	}
	return nil
}

// IsNotEmpty checks if the provided slice is not empty.
// It returns an error if the slice is empty, using the provided message.
func IsNotEmpty[T any](s []T, msg string) error {
	if len(s) == 0 {
		return createError(msg)
	}
	return nil
}

// IsNotEmptyf checks if the provided slice is not empty with formatted message.
// It returns an error if the slice is empty, formatting the message with the provided arguments.
func IsNotEmptyf[T any](s []T, format string, args ...any) error {
	if len(s) == 0 {
		return createErrorf(format, args...)
	}
	return nil
}

// KeyNotInMap checks if the given key does not exist in the provided map.
// It returns an error if the key exists in the map, using the provided message.
func KeyNotInMap[K comparable, V any](key K, m map[K]V, msg string) error {
	if _, exists := m[key]; exists {
		return createError(msg)
	}
	return nil
}

// KeyNotInMapf checks if the given key does not exist in the provided map with formatted message.
// It returns an error if the key exists in the map, formatting the message with the provided arguments.
func KeyNotInMapf[K comparable, V any](key K, m map[K]V, format string, args ...any) error {
	if _, exists := m[key]; exists {
		return createErrorf(format, args...)
	}
	return nil
}
