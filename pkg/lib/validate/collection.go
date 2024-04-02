package validate

func IsEmpty[T any](s []T) bool {
	return len(s) == 0
}

func IsNotEmpty[T any](s []T) bool {
	return !IsEmpty(s)
}
