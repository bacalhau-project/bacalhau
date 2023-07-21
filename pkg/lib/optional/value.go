package optional

// ValueOptional is an implementation of Optional that holds a value of type T.
type ValueOptional[T any] struct {
	value T
}

func (o ValueOptional[T]) IsPresent() bool {
	return true
}

func (o ValueOptional[T]) Get() (T, error) {
	return o.value, nil
}

func (o ValueOptional[T]) GetOrDefault(defaultValue T) T {
	return o.value
}

// compile time check that ValueOptional[T] implements Optional[T]
var _ Optional[string] = (*ValueOptional[string])(nil)
