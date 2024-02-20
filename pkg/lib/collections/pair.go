package collections

import "fmt"

// Pair is a generic structure that holds two values of any type.
type Pair[L any, R any] struct {
	Left  L
	Right R
}

// NewPair creates a new Pair with the given values.
func NewPair[L any, R any](left L, right R) Pair[L, R] {
	return Pair[L, R]{Left: left, Right: right}
}

// String returns a string representation of the Pair.
func (p Pair[L, R]) String() string {
	return fmt.Sprintf("(%v, %v)", p.Left, p.Right)
}
