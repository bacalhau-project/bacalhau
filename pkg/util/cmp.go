package util

import "golang.org/x/exp/constraints"

// Compare exists as a temporary measure to provide a simple cmp()
// implementation for any ordered types.  This was implemented to
// enable us to handle the exp/slices update to SortFunc() which
// switched from expecting a less() function to a cmp() function.
//
// Once we upgrade to Go 1.21 we can remove this and use the builtin
// cmp package - see https://pkg.go.dev/cmp
type Compare[T constraints.Ordered] struct{}

// Cmp returns -1 if a < b, 0 if a == b, and 1 if a > b.
func (c Compare[T]) Cmp(a, b T) int {
	if a < b {
		return -1
	} else if a > b {
		return 1
	}
	return 0
}

// CmpRev returns -1 if a > b, 0 if a == b, and 1 if a < b.
// This is the reverse of Cmp
func (c Compare[T]) CmpRev(a, b T) int {
	if a < b {
		return 1
	} else if a > b {
		return -1
	}
	return 0
}
