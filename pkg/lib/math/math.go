package math

import (
	"github.com/samber/lo"
	"golang.org/x/exp/constraints"
)

type Number interface {
	constraints.Integer | constraints.Float
}

// Min function implementation
func Min[T constraints.Ordered](item T, items ...T) T {
	return lo.Min(append(items, item))
}

// Max function implementation
func Max[T constraints.Ordered](item T, items ...T) T {
	return lo.Max(append(items, item))
}

// Abs function implementation that supports integers, and not only floats
// like in the standard library
func Abs[T Number](number T) T {
	if number < 0 {
		return -number
	}
	return number
}
