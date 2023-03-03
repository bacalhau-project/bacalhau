package logger

import "fmt"

func ToStringer[T any](t T, f func(t T) string) fmt.Stringer {
	return stringerHelper[T]{
		f: f,
		t: t,
	}
}

func ToSliceStringer[T any](ts []T, f func(t T) string) []fmt.Stringer {
	stringers := make([]fmt.Stringer, 0, len(ts))
	for _, t := range ts {
		stringers = append(stringers, ToStringer(t, f))
	}
	return stringers
}

type stringerHelper[T any] struct {
	t T
	f func(t T) string
}

func (s stringerHelper[T]) String() string {
	return s.f(s.t)
}
