package math

import (
	"github.com/samber/lo"
	"golang.org/x/exp/constraints"
)

// Min function implementation
func Min[T constraints.Ordered](item T, items ...T) T {
	return lo.Min(append(items, item))
}

// Max function implementation
func Max[T constraints.Ordered](item T, items ...T) T {
	return lo.Max(append(items, item))
}
