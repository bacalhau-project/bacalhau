package models

// Copyable is an interface for types that can be copied
type Copyable[T any] interface {
	Copy() T
}

// Normalizable is an interface for types that can be normalized
// (e.g. empty maps are converted to nil)
type Normalizable interface {
	Normalize()
}

// Validatable is an interface for types that can be validated
type Validatable interface {
	Validate() error
}
