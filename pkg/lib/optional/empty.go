package optional

import "errors"

type EmptyOptional[T any] struct {
}

func (o EmptyOptional[T]) IsPresent() bool {
	return false
}

func (o EmptyOptional[T]) Get() (T, error) {
	return *new(T), errors.New("get() called on empty optional")
}

func (o EmptyOptional[T]) GetOrDefault(defaultValue T) T {
	return defaultValue
}

// compile time check that EmptyOptional[T] implements ValueOptional[T]
var _ Optional[string] = (*EmptyOptional[string])(nil)
