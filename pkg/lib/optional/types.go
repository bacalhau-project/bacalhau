package optional

// Optional represents an optional value that can either hold a value of type T or be empty.
// Useful when the empty value of T is a valid value, and you need to differentiate between
// the empty value and the absence of a value.
type Optional[T any] interface {
	// IsPresent checks if the Optional has a value present.
	IsPresent() bool
	// Get retrieves the value stored in the Optional.
	// If the Optional is empty, it returns an error.
	Get() (T, error)
	// GetOrDefault retrieves the value stored in the Optional,
	// or returns the specified defaultValue if the Optional is empty.
	GetOrDefault(defaultValue T) T
}

// New creates a new ValueOptional with the specified value.
func New[T any](value T) ValueOptional[T] {
	return ValueOptional[T]{value: value}
}

func Empty[T any]() EmptyOptional[T] {
	return EmptyOptional[T]{}
}
